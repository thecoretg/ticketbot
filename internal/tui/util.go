package tui

import (
	"sort"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
)

func boolToIcon(b bool) string {
	i := "✗"
	if b {
		i = "✓"
	}

	return i
}

func shortenSourceType(s string) string {
	switch s {
	case "person":
		return "p"
	case "room":
		return "r"
	default:
		return "?"
	}
}

func sortBoards(boards []models.Board) {
	sort.SliceStable(boards, func(i, j int) bool {
		return boards[i].Name < boards[j].Name
	})
}

func sortRecips(recips []models.WebexRecipient) {
	sort.SliceStable(recips, func(i, j int) bool {
		return recips[i].Name < recips[j].Name
	})
}

func sortRules(rules []models.NotifierRuleFull) {
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].BoardName < rules[j].BoardName
	})
}

func isValidDate(input string) bool {
	_, err := time.Parse("2006-01-02", input)
	return err == nil
}
