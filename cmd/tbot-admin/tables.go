package main

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	headerStyle = lipgloss.NewStyle().Align(lipgloss.Center)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1).Align(lipgloss.Center)
)

func webexRoomsToTable(rooms []models.WebexRecipient) {
	sort.SliceStable(rooms, func(i, j int) bool {
		return rooms[i].Name < rooms[j].Name
	})

	t := defaultTable()
	t.Headers("ID", "TYPE", "NAME")
	for _, r := range rooms {
		if r.Name != "Empty Title" {
			t.Row(strconv.Itoa(r.ID), string(r.Type), r.Name)
		}
	}

	fmt.Println(t)
}

func apiUsersTable(users []models.APIUser) {
	sort.SliceStable(users, func(i, j int) bool {
		return users[i].EmailAddress < users[j].EmailAddress
	})

	t := defaultTable()
	t.Headers("ID", "EMAIL", "CREATED ON")
	for _, u := range users {
		t.Row(strconv.Itoa(u.ID), u.EmailAddress, u.CreatedOn.Format("2006-01-02"))
	}

	fmt.Println(t)
}

func apiKeysTable(keys []models.APIKey) {
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].UserID < keys[j].UserID
	})

	t := defaultTable()
	t.Headers("USER ID", "KEY ID", "CREATED ON")
	for _, k := range keys {
		t.Row(strconv.Itoa(k.UserID), strconv.Itoa(k.ID), k.CreatedOn.Format("2006-01-02"))
	}

	fmt.Println(t)
}

func cwBoardsToTable(boards []models.Board) {
	sort.SliceStable(boards, func(i, j int) bool {
		return boards[i].Name < boards[j].Name
	})

	t := defaultTable()
	t.Headers("ID", "NAME")
	for _, b := range boards {
		t.Row(strconv.Itoa(b.ID), b.Name)
	}

	fmt.Println(t)
}

func notifierRulesTable(notifiers []models.NotifierRuleFull) {
	t := defaultTable()
	t.Headers("ID", "ENABLED", "BOARD", "RECIPIENT")
	for _, n := range notifiers {
		t.Row(strconv.Itoa(n.ID), boolToIcon(n.Enabled), n.BoardName, fmt.Sprintf("%s (%s)", n.RecipientName, n.RecipientType))
	}

	fmt.Println(t)
}

func userForwardsTable(forwards []models.NotifierForwardFull) {
	t := defaultTable()
	t.Headers("ID", "ENABLED", "START", "END", "KEEP COPY", "SOURCE", "DESTINATION")
	for _, uf := range forwards {
		sd := "NA"
		ed := "NA"
		if uf.StartDate != nil {
			sd = uf.StartDate.Format("2006-01-02")
		}
		if uf.EndDate != nil {
			ed = uf.EndDate.Format("2006-01-02")
		}

		t.Row(
			strconv.Itoa(uf.ID),
			boolToIcon(uf.Enabled),
			sd,
			ed,
			boolToIcon(uf.UserKeepsCopy),
			fmt.Sprintf("%s (%s)", uf.SourceName, uf.SourceType),
			fmt.Sprintf("%s (%s)", uf.DestinationName, uf.DestinationType),
		)
	}

	fmt.Println(t)
}

func defaultTable() *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			} else {
				return cellStyle
			}
		})
}

func boolToIcon(b bool) string {
	i := "✗"
	if b {
		i = "✓"
	}

	return i
}
