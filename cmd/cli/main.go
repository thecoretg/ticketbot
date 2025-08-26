package main

import (
	"fmt"
	"os"

	"github.com/thecoretg/ticketbot/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Printf("An error occured: %v", err)
		os.Exit(1)
	}
}
