#!/bin/bash

# build binary
GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/lambda/main.go
zip lambda.zip bootstrap
aws lambda update-function-code --function-name ticketbot \
  --zip-file fileb://lambda.zip \
  --no-cli-pager

rm lambda.zip bootstrap