package ticketbot

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log/slog"
	"net/http"
	"os"
	"tctg-automation/internal/ticketbot/db"
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
	dbHandler   *db.Handler

	rootUrl string
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

	r, err := s.newRouter(os.Getenv("TICKETBOT_EXIT_ON_ERROR") == "1")
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
	cw, err := connectwise.NewClientFromAWS(ctx, h, nil, s, cwCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating connectwise client via AWS: %w", err)
	}

	slog.Debug("initializing Webex client with AWS creds")
	w, err := webex.NewClientFromAWS(ctx, h, s, webexCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating webex client via AWS: %w", err)
	}

	dbHandler, err := db.InitDB(os.Getenv("TICKETBOT_DB_CONN"))
	if err != nil {
		return nil, fmt.Errorf("initializing db: %w", err)
	}

	return &server{
		cwClient:    cw,
		webexClient: w,
		dbHandler:   dbHandler,

		rootUrl: addr,
	}, nil
}

func (s *server) newRouter(exitOnError bool) (*gin.Engine, error) {
	ctx := context.Background()

	if err := s.loadInitialData(ctx); err != nil {
		return nil, fmt.Errorf("loading initial data: %w", err)
	}

	if err := s.initiateTicketWebhook(ctx); err != nil {
		return nil, fmt.Errorf("initiating tickets webhook: %w", err)
	}

	r := gin.Default()
	r.Use(requireValidCWSignature())
	r.Use(ErrorHandler(exitOnError))

	r.GET("/boards/:board_id", s.getBoard)
	r.PUT("/boards/:board_id/notify", s.processBoardSettingsPayload)
	r.POST("/tickets", s.processTicketPayload)
	r.POST("/companies", s.processCompanyPayload)
	r.POST("/contacts", s.processContactPayload)
	r.POST("/members", s.processMemberPayload)

	return r, nil
}
