package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"log/slog"
	"os"
	"tctg-automation/internal/ticketbot"
)

var ginLambda *ginadapter.GinLambda

func init() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

func main() {
	slog.Info("Starting ticketbot server...if running via Lambda, this is a cold start.")
	ctx := context.Background()
	s, err := ticketbot.NewServer(ctx)
	if err != nil {
		slog.Error("creating server config", "error", err)
		os.Exit(1)
	}

	r, err := s.NewRouter()
	if err != nil {
		slog.Error("creating router", "error", err)
		os.Exit(1)
	}

	// Only run as lambda if detected as lambda - for easy testing
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		ginLambda = ginadapter.New(r)
		lambda.Start(Handler)
	} else {
		// Otherwise, just run as a normal Gin server locally
		err := r.Run(":80")
		if err != nil {
			slog.Error("running server", "error", err)
			os.Exit(1)
		}
	}
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}
