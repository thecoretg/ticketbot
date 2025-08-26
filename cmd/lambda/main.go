package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/thecoretg/ticketbot/ticketbot"
)

var ginLambda *ginadapter.GinLambda

func init() {
	slog.Info("lambda cold start")
	ctx := context.Background()
	cfg, err := ticketbot.InitCfg()
	if err != nil {
		log.Fatalf("initializing config: %v", err)
	}

	s, err := ticketbot.NewServer(ctx, cfg, true)
	if err != nil {
		log.Fatalf("initializing server: %v", err)
	}

	ginLambda = ginadapter.New(s.GinEngine)
}

func main() {
	lambda.Start(LambdaHandler)
}

func LambdaHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}
