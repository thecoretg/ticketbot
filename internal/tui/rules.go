package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/thecoretg/ticketbot/internal/models"
)

type rulesModelKeys struct {
	up   key.Binding
	down key.Binding
}

var defaultRulesKeys = rulesModelKeys{
	up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "up"),
	),
	down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "down"),
	),
}

type rulesModel struct {
	keys          rulesModelKeys
	gotDimensions bool
	width         int
	height        int
	rulesLoaded   bool
	selected      int

	rules []models.NotifierRuleFull
}

func newRulesModel() *rulesModel {
	return &rulesModel{
		keys:  defaultRulesKeys,
		rules: []models.NotifierRuleFull{},
	}
}

func (rm *rulesModel) Init() tea.Cmd {
	return nil
}

func (rm *rulesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		rm.width = msg.Width
		rm.height = msg.Height
		rm.gotDimensions = true
	case gotRulesMsg:
		rm.rules = msg.rules
		rm.rulesLoaded = true
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, rm.keys.up):
			top := len(rm.rules) - 1
			if rm.selected == 0 {
				rm.selected = top
			} else {
				rm.selected--
			}
		case key.Matches(msg, rm.keys.down):
			top := len(rm.rules) - 1
			if rm.selected < top {
				rm.selected++
			} else {
				rm.selected = 0
			}
		}
	}

	return rm, nil
}

func (rm *rulesModel) View() string {
	if !rm.gotDimensions {
		return "Initializing..."
	}

	if !rm.rulesLoaded {
		return "Loading rules..."
	}

	return rm.rulesTable(rm.width)
}

func (rm *rulesModel) rulesTable(w int) string {
	t := table.New().
		Border(border).
		Width(w).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch row {
			case table.HeaderRow:
				return headerStyle
			case rm.selected:
				return selectedStyle
			default:
				return unselectedStyle
			}
		}).
		Headers("ENABLED", "BOARD", "RECIPIENT").
		Rows(rulesToRows(rm.rules)...)

	return t.Render()
}

func rulesToRows(rules []models.NotifierRuleFull) [][]string {
	var rows [][]string
	for _, r := range rules {
		rows = append(rows, []string{boolToIcon(r.Enabled), r.BoardName, r.RecipientName})
	}

	return rows
}
