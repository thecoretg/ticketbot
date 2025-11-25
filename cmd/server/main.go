package main

import (
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		slog.Error("running the server", "error", err)
	}
}
