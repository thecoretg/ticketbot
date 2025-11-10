package main

import (
	"fmt"

	"github.com/thecoretg/ticketbot/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Printf("An error occured: %v\n", err)
	}
}
