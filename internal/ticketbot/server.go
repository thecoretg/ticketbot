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
	dbHandler   *DBHandler

	rootUrl string
}

func Run() error {
	if err := godotenv.Load(); err != nil {
		slog.Warn("loading .env file", "error", err)
	}
	setLogger(os.Getenv("TICKETBOT_DEBUG"))

	ctx := context.Background()
	s, err := newServer(ctx, os.Getenv("TICKETBOT_ROOT_URL"))
	if err != nil {
		slog.Error("creating server config", "error", err)
		return fmt.Errorf("error creating server config: %w", err)
	}

	r, err := s.newRouter()
	if err != nil {
		slog.Error("creating router", "error", err)
		return fmt.Errorf("error creating router: %w", err)
	}

	if err := r.Run(":80"); err != nil {
		slog.Error("running server", "error", err)
		return fmt.Errorf("error running server: %w", err)
	}

	return nil
}

func setLogger(e string) {
	level := slog.LevelInfo
	if e == "1" || e == "true" {
		slog.Info("----- DEBUG ON -----")
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

func newServer(ctx context.Context, addr string) (*server, error) {
	if addr == "" {
		return nil, fmt.Errorf("TICKETBOT_ROOT_URL cannot be blank")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating aws default config: %w", err)
	}

	s := ssm.NewFromConfig(cfg)
	h := http.DefaultClient
	cw, err := connectwise.NewClientFromAWS(ctx, h, nil, s, cwCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating connectwise client via AWS: %w", err)
	}

	w, err := webex.NewClientFromAWS(ctx, h, s, webexCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating webex client via AWS: %w", err)
	}

	dbHandler, err := InitDB(os.Getenv("TICKETBOT_DB_CONN"))
	if err != nil {
		return nil, fmt.Errorf("initializing db: %w", err)
	}

	server := &server{
		cwClient:    cw,
		webexClient: w,
		dbHandler:   dbHandler,

		rootUrl: addr,
	}

	return server, nil
}

func (s *server) newRouter() (*gin.Engine, error) {
	ctx := context.Background()
	if err := s.initiateWebhook(ctx); err != nil {
		return nil, fmt.Errorf("initiating tickets webhook: %w", err)
	}

	r := gin.Default()
	r.POST("/tickets", s.processTicketPayload)

	return r, nil
}
