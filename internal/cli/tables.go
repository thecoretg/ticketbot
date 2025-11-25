package cli

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/oldserver"
)

var (
	headerStyle = lipgloss.NewStyle().Align(lipgloss.Center)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
)

func webexRoomsToTable(rooms []db.WebexRoom) {
	sort.SliceStable(rooms, func(i, j int) bool {
		return rooms[i].Name < rooms[j].Name
	})

	t := defaultTable()
	t.Headers("ID", "TYPE", "NAME")
	for _, r := range rooms {
		if r.Name != "Empty Title" {
			t.Row(strconv.Itoa(r.ID), r.Type, r.Name)
		}
	}

	fmt.Println(t)
}

func cwBoardsToTable(boards []db.CwBoard) {
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

func notifiersTable(notifiers []oldserver.Notifier) {
	t := defaultTable()
	t.Headers("ID", "ROOM", "BOARD", "NOTIFY")
	for _, n := range notifiers {
		t.Row(strconv.Itoa(n.ID), n.WebexRoom.Name, n.CWBoard.Name, fmt.Sprintf("%v", n.NotifyEnabled))
	}

	fmt.Println(t)
}

func defaultTable() *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).BorderTop(false).BorderBottom(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			} else {
				return cellStyle
			}
		})
}
