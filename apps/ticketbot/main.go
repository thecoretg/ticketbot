package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"log"
	"tctg-automation/internal/ticketbot"
)

var ginLambda *ginadapter.GinLambda

func init() {
	g, err := ticketbot.GetGinEngine()
	if err != nil {
		log.Fatalf(err.Error())
	}

	ginLambda = ginadapter.New(g)
}

func main() {
	lambda.Start(LambdaHandler)
}

func LambdaHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, req)
}
