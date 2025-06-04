package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"os"
	"tctg-automation/internal/ticketbot"
)

var ginLambda *ginadapter.GinLambda

func main() {
	fmt.Println("Starting ticketbot server...if running via Lambda, this is a cold start.")
	ctx := context.Background()
	s, err := ticketbot.NewServer(ctx)
	if err != nil {
		panic(err)
	}

	r, err := s.NewRouter()
	if err != nil {
		panic(err)
	}

	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		ginLambda = ginadapter.New(r)
		lambda.Start(Handler)
	} else {
		r.Run(":8080")
	}
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}
