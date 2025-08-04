update-server:
	GOOS=linux GOARCH=amd64 go build -o bin/ticketbot .
	scp bin/ticketbot ec2-user@ticketbot.thecoretg.com:ticketbot