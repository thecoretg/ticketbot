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
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
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

func notifierRulesTable(notifiers []models.NotifierRule) {
	t := defaultTable()
	t.Headers("ID", "ROOM", "BOARD", "NOTIFY")
	for _, n := range notifiers {
		t.Row(strconv.Itoa(n.ID), strconv.Itoa(n.WebexRoomID), strconv.Itoa(n.CwBoardID), fmt.Sprintf("%v", n.NotifyEnabled))
	}

	fmt.Println(t)
}

func userForwardsTable(forwards []models.NotifierForward) {
	t := defaultTable()
	t.Headers("ID", "SRC ID", "DEST ID", "START DATE", "END DATE", "ENABLED", "USER KEEPS COPY")
	for _, uf := range forwards {
		ed := "NA"
		if uf.EndDate != nil {
			ed = uf.EndDate.Format("2006-01-02")
		}

		t.Row(
			strconv.Itoa(uf.ID),
			strconv.Itoa(uf.SourceID),
			strconv.Itoa(uf.DestID),
			uf.StartDate.Format("2006-01-02"),
			ed,
			fmt.Sprintf("%v", uf.Enabled),
			fmt.Sprintf("%v", uf.UserKeepsCopy),
		)
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
