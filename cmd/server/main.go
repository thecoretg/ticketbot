package main

import (
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/newserver"
)

func main() {
	if err := newserver.Run(); err != nil {
		slog.Error("running the server", "error", err)
	}
}
