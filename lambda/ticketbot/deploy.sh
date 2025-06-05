GOOS=linux GOARCH=arm64 go build -o bootstrap main.go
zip lambda-function.zip bootstrap
rm bootstrap
aws lambda update-function-code \
  --function-name ticketbot \
  --zip-file fileb://lambda-function.zip \
  --publish

rm -rf lambda-function.zip