package main

import (
	"log"
	"tctg-automation/internal/ticketbot"
)

func main() {
	if err := ticketbot.Run(); err != nil {
		log.Fatal(err)
	}
}
