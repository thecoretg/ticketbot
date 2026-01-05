package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thecoretg/ticketbot/internal/models"
)

type fwdsModel struct {
	gotDimensions bool
	width         int
	height        int
	fwdsLoaded    bool
	table         table.Model

	fwds []models.NotifierForwardFull
}

func newFwdsModel() *fwdsModel {
	h := help.New()
	h.Styles.ShortDesc = helpStyle
	h.Styles.ShortKey = helpStyle
	return &fwdsModel{
		fwds:  []models.NotifierForwardFull{},
		table: newTable(),
	}
}

func (fm *fwdsModel) Init() tea.Cmd {
	return nil
}

func (fm *fwdsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fm.width = msg.Width
		fm.height = msg.Height
		fm.gotDimensions = true
	case gotFwdsMsg:
		fm.fwds = msg.fwds
		fm.fwdsLoaded = true
		fm.table.SetRows(fwdsToRows(fm.fwds))
	case resizeModelsMsg:
		fm.setTableDimensions(msg.w, msg.h)
	}

	var cmd tea.Cmd
	fm.table, cmd = fm.table.Update(msg)

	return fm, cmd
}

func (fm *fwdsModel) View() string {
	if !fm.gotDimensions {
		return "Initializing..."
	}

	if !fm.fwdsLoaded {
		return "Loading forwards..."
	}

	return fm.table.View()
}

func (fm *fwdsModel) setTableDimensions(w, h int) {
	t := &fm.table
	enableW := 8
	keepW := 8
	datesW := 13
	srcW := 25
	remainingW := w - enableW - datesW - keepW - srcW
	destW := remainingW
	t.SetColumns(
		[]table.Column{
			{Title: "ENABLED", Width: enableW},
			{Title: "KEEP", Width: keepW},
			{Title: "DATES", Width: datesW},
			{Title: "SOURCE", Width: srcW},
			{Title: "DESTINATION", Width: destW},
		},
	)
	t.SetHeight(h)
}

func fwdsToRows(fwds []models.NotifierForwardFull) []table.Row {
	var rows []table.Row
	for _, f := range fwds {
		src := fmt.Sprintf("%s (%s)", f.SourceName, shortenSourceType(f.SourceType))
		dst := fmt.Sprintf("%s (%s)", f.DestinationName, shortenSourceType(f.DestinationType))
		sd := f.StartDate.Format("01-02")
		ed := "N/A"
		if f.EndDate != nil {
			ed = f.EndDate.Format("01-02")
		}
		dr := fmt.Sprintf("%s - %s", sd, ed)

		rows = append(rows, []string{
			boolToIcon(f.UserKeepsCopy),
			boolToIcon(f.UserKeepsCopy),
			dr,
			src,
			dst,
		})
	}

	return rows
}
