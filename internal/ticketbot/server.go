package ticketbot

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"net/http"
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

	Boards []boardSetting `json:"boards"`
}

func (s *Server) NewRouter() (*gin.Engine, error) {
	r := gin.Default()

	r.GET("/boards", s.listBoardsEndpoint)
	r.POST("/boards", s.addOrUpdateBoardEndpoint)
	r.DELETE("/boards/:board_id", s.deleteBoardEndpoint)

	return r, nil
}

func NewServer(ctx context.Context) (*Server, error) {
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
