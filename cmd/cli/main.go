package main

import (
	"fmt"
	"github.com/thecoretg/ticketbot/cli"
	"os"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Printf("An error occured: %v", err)
		os.Exit(1)
	}
}
