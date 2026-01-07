package tui

import (
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/thecoretg/ticketbot/internal/models"
)

func boardsToFormOpts(boards []models.Board) []huh.Option[models.Board] {
	var opts []huh.Option[models.Board]
	for _, b := range boards {
		o := huh.NewOption(b.Name, b)
		opts = append(opts, o)
	}

	return opts
}

func recipsToFormOpts(recips, exclude []models.WebexRecipient) []huh.Option[models.WebexRecipient] {
	var opts []huh.Option[models.WebexRecipient]
	for _, r := range recips {
		if slices.Contains(exclude, r) {
			continue
		}

		o := huh.NewOption(r.Name, r)
		opts = append(opts, o)
	}

	return opts
}
