package tui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/thecoretg/ticketbot/internal/models"
)

type fwdsModelKeys struct {
	up   key.Binding
	down key.Binding
}

var defaultFwdsKeys = fwdsModelKeys{
	up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "up"),
	),
	down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "down"),
	),
}

type fwdsModel struct {
	keys          fwdsModelKeys
	gotDimensions bool
	width         int
	height        int
	fwdsLoaded    bool
	selected      int

	fwds []models.NotifierForwardFull
}

func newFwdsModel() *fwdsModel {
	return &fwdsModel{
		keys: defaultFwdsKeys,
		fwds: []models.NotifierForwardFull{},
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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, fm.keys.up):
			top := len(fm.fwds) - 1
			if fm.selected == 0 {
				fm.selected = top
			} else {
				fm.selected--
			}
		case key.Matches(msg, fm.keys.down):
			top := len(fm.fwds) - 1
			if fm.selected < top {
				fm.selected++
			} else {
				fm.selected = 0
			}
		}
	}

	return fm, nil
}

func (fm *fwdsModel) View() string {
	if !fm.gotDimensions {
		return "Initializing..."
	}

	if !fm.fwdsLoaded {
		return "Loading forwards..."
	}

	return fm.fwdsTable(fm.width)
}

func (fm *fwdsModel) fwdsTable(w int) string {
	t := table.New().
		Border(border).
		Width(w).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch row {
			case table.HeaderRow:
				return headerStyle
			case fm.selected:
				return selectedStyle
			default:
				return unselectedStyle
			}
		}).
		Headers("ENABLED", "START", "END", "KEEP COPY", "SOURCE", "DESTINATION").
		Rows(fwdsToRows(fm.fwds)...)

	return t.Render()
}

func fwdsToRows(fwds []models.NotifierForwardFull) [][]string {
	var rows [][]string
	for _, f := range fwds {
		src := fmt.Sprintf("%s (%s)", f.SourceName, f.SourceType)
		dst := fmt.Sprintf("%s (%s)", f.DestinationName, f.DestinationType)
		sd := f.StartDate.Format("2006-01-02")
		ed := "N/A"
		if f.EndDate != nil {
			ed = f.EndDate.Format("2006-01-02")
		}

		rows = append(rows, []string{
			boolToIcon(f.UserKeepsCopy),
			sd,
			ed,
			strconv.FormatBool(f.UserKeepsCopy),
			src,
			dst,
		})
	}

	return rows
}
