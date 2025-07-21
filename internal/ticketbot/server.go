package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log/slog"
	"net/http"
	"os"
	"tctg-automation/internal/ticketbot/store"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"
)

const (
	// name of creds parameter in AWS systems manager
	cwCredsParam    = "/connectwise/creds/ticketbot"
	webexCredsParam = "/webex/keys/ticketbot"
)

type server struct {
	cwClient    *connectwise.Client
	webexClient *webex.Client
	dataStore   store.Store

	webexSecret       string
	webexBotEmail     string
	initialAdminEmail string
	exitOnError       bool
	rootUrl           string
}

func Run() error {
	slog.Info("loading environment variables...")
	if err := godotenv.Load(); err != nil {
		fmt.Println("no .env file present, defaulting to environment")
	}
	if err := setLogger(os.Getenv("TICKETBOT_DEBUG"), os.Getenv("TICKETBOT_LOG_TO_FILE") == "1"); err != nil {
		return fmt.Errorf("error setting logger: %w", err)
	}

	ctx := context.Background()
	s, err := newServer(ctx, os.Getenv("TICKETBOT_ROOT_URL"))
	if err != nil {
		return fmt.Errorf("error creating server config: %w", err)
	}

	r, err := s.newRouter()
	if err != nil {
		return fmt.Errorf("error creating router: %w", err)
	}

	if err := r.Run(":80"); err != nil {
		return fmt.Errorf("error running server: %w", err)
	}

	return nil
}

func newServer(ctx context.Context, addr string) (*server, error) {
	if addr == "" {
		return nil, fmt.Errorf("TICKETBOT_ROOT_URL cannot be blank")
	}

	slog.Debug("initializing AWS systems manager client")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating aws default config: %w", err)
	}

	s := ssm.NewFromConfig(cfg)
	h := http.DefaultClient
	slog.Debug("initializing CW client with AWS creds")
	cw, err := connectwise.NewClientFromAWS(ctx, s, cwCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating connectwise client via AWS: %w", err)
	}

	slog.Debug("initializing Webex client with AWS creds")
	w, err := webex.NewClientFromAWS(ctx, h, s, webexCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating webex client via AWS: %w", err)
	}

	webexSecret := os.Getenv("TICKETBOT_WEBEX_SECRET")
	if webexSecret == "" {
		return nil, errors.New("webex secret cannot be empty")
	}

	webexBotEmail := os.Getenv("TICKETBOT_BOT_EMAIL")
	if webexBotEmail == "" {
		return nil, errors.New("webex bot email cannot be empty")
	}

	exitOnError := os.Getenv("TICKETBOT_EXIT_ON_ERROR") == "1"
	dataStore := store.NewInMemoryStore()
	return &server{
		cwClient:    cw,
		webexClient: w,
		dataStore:   dataStore,

		webexSecret:   webexSecret,
		webexBotEmail: webexBotEmail,
		exitOnError:   exitOnError,
		rootUrl:       addr,
	}, nil
}

func (s *server) newRouter() (*gin.Engine, error) {
	if err := s.initiateCWHooks(); err != nil {
		return nil, fmt.Errorf("initiating connectwise hooks: %w", err)
	}

	r := gin.Default()
	s.addTicketGroup(r)
	return r, nil
}
