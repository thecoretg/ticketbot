package main

import (
	"log"
	"tctg-automation/internal/ticketbot"
)

func main() {
	if err := ticketbot.RunServer(); err != nil {
		log.Fatal(err)
	}
}
