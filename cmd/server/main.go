package main

import (
	"embed"
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/server"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

func main() {
	if err := server.Run(embeddedMigrations); err != nil {
		slog.Error("an error occured running the server", "error", err)
	}
}
