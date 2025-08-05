update-server:
	GOOS=linux GOARCH=amd64 go build -o bin/ticketbot .
	scp bin/ticketbot ubuntu@ticketbot.thecoretg.com:ticketbot

run:
	go build -o bin/ticketbot . && bin/ticketbot