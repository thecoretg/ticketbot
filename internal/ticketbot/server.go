package ticketbot

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"net/http"
	"tctg-automation/pkg/aws"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"
)

const (
	// name of creds parameter in AWS systems manager
	cwCredsParam    = "/connectwise/creds/ticketbot"
	webexCredsParam = "/webex/keys/ticketbot"
)

type Client struct {
	cwClient    *connectwise.Client
	webexClient *webex.Client
	db          *dynamodb.DynamoDB
}

func NewRouter() (*gin.Engine, error) {

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	return r, nil
}

func newClient(ctx context.Context) (*Client, error) {
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

	db := aws.NewDBConn()

	return &Client{
		cwClient:    cw,
		webexClient: w,
		db:          db,
	}, nil
}

func getCwClient() (*connectwise.Client, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating aws default config: %w", err)
	}
	s := ssm.NewFromConfig(cfg)

	c, err := connectwise.GetCredsFromAWS(ctx, s, cwCredsParam, true)
	if err != nil {
		return nil, fmt.Errorf("getting credentials from AWS: %w", err)
	}

	return connectwise.NewClient(*c, http.DefaultClient), nil
}
