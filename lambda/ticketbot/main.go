package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"os"
	"tctg-automation/internal/ticketbot"
)

func main() {
	r, err := ticketbot.NewRouter()
	if err != nil {
		panic(err)
	}

	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		ginLambda := ginadapter.New(r)
		lambda.Start(ginLambda.Proxy)
	} else {
		r.Run(":8080")
	}
}
