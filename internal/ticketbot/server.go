package ticketbot

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"tctg-automation/pkg/amz"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"
)

const (
	// name of creds parameter in AWS systems manager
	cwCredsParam    = "/connectwise/creds/ticketbot"
	webexCredsParam = "/webex/keys/ticketbot"
)

type Server struct {
	cwClient    *connectwise.Client
	webexClient *webex.Client
	db          *dynamodb.DynamoDB

	// map of user identifiers to their emails
	users   map[string]string
	rootUrl string

	Boards []boardSetting `json:"boards"`
}

func (s *Server) NewRouter() (*gin.Engine, error) {
	ctx := context.Background()
	if err := s.initiateWebhook(ctx); err != nil {
		return nil, fmt.Errorf("initiating tickets webhook: %w", err)
	}

	r := gin.Default()

	ticketbot := r.Group("/ticketbot")
	{
		ticketbot.GET("/boards", s.listBoardsEndpoint)
		ticketbot.POST("/boards", s.addOrUpdateBoardEndpoint)
		ticketbot.DELETE("/boards/:board_id", s.deleteBoardEndpoint)
		ticketbot.POST("/tickets", s.handleTicketEndpoint)
	}
	return r, nil
}

func NewServer(ctx context.Context) (*Server, error) {
	u := os.Getenv("TICKETBOT_ROOT_URL")
	if u == "" {
		return nil, fmt.Errorf("TICKETBOT_ROOT_URL cannot be blank")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating aws default config: %w", err)
	}

	s := ssm.NewFromConfig(cfg)
	h := http.DefaultClient
	cw, err := connectwise.NewClientFromAWS(ctx, h, s, cwCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating connectwise client via AWS: %w", err)
	}

	w, err := webex.NewClientFromAWS(ctx, h, s, webexCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("creating webex client via AWS: %w", err)
	}

	db := amz.NewDBConn()

	server := &Server{
		cwClient:    cw,
		webexClient: w,
		db:          db,
		Boards:      []boardSetting{},
		rootUrl:     u,
		users:       make(map[string]string),
	}

	if err := server.refreshBoards(); err != nil {
		return nil, fmt.Errorf("refreshing boards: %w", err)
	}

	return server, nil
}

func (s *Server) refreshBoards() error {
	var err error
	s.Boards, err = s.listBoards()
	if err != nil {
		return fmt.Errorf("refreshing boards: %w", err)
	}

	return nil
}
