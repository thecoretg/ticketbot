package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

type (
	rulesModel struct {
		SDKClient   *sdk.Client
		rulesLoaded bool
		table       table.Model
		form        *huh.Form
		formResult  *rulesFormResult
		status      rulesModelStatus

		availWidth  int
		availHeight int
		rules       []models.NotifierRuleFull
	}

	ruleFormDataMsg struct {
		boards []models.Board
		recips []models.WebexRecipient
	}

	updateRmStatusMsg struct{ status rulesModelStatus }
)
type rulesModelStatus int

const (
	rmStatusInitializing rulesModelStatus = iota
	rmStatusTable
	rmStatusLoadingData
	rmStatusEntry
	rmStatusRefreshing
)

type rulesFormResult struct {
	board models.Board
	recip models.WebexRecipient
}

func newRulesModel(cl *sdk.Client) *rulesModel {
	return &rulesModel{
		SDKClient:  cl,
		rules:      []models.NotifierRuleFull{},
		table:      newTable(),
		formResult: &rulesFormResult{},
	}
}

func (rm *rulesModel) Init() tea.Cmd {
	return nil
}

func (rm *rulesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, allKeys.newItem):
			return rm, tea.Batch(updateRulesStatus(rmStatusLoadingData), rm.prepareForm())
		case key.Matches(msg, allKeys.deleteItem):
			rule := rm.rules[rm.table.Cursor()]
			return rm, tea.Batch(updateRulesStatus(rmStatusRefreshing), rm.deleteRule(rule.ID))
		}

	case resizeModelsMsg:
		rm.availWidth = msg.w
		rm.availHeight = msg.h
		rm.setModuleDimensions()

	case gotRulesMsg:
		rm.rules = msg.rules
		rm.rulesLoaded = true
		return rm, tea.Batch(updateRulesStatus(rmStatusTable), rm.setRows())

	case ruleFormDataMsg:
		rm.formResult = &rulesFormResult{}
		rm.form = ruleEntryForm(msg.boards, msg.recips, rm.formResult, rm.availHeight)
		return rm, tea.Batch(updateRulesStatus(rmStatusEntry), rm.form.Init())

	case updateRmStatusMsg:
		rm.status = msg.status
	}

	switch rm.status {
	case rmStatusEntry:

		var cmds []tea.Cmd

		form, cmd := rm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			rm.form = f
		}

		cmds = append(cmds, cmd)
		switch rm.form.State {
		case huh.StateAborted:
			cmds = append(cmds, updateRulesStatus(rmStatusTable))

		case huh.StateCompleted:
			res := rm.formResult
			rule := &models.NotifierRule{
				CwBoardID:        res.board.ID,
				WebexRecipientID: res.recip.ID,
				NotifyEnabled:    true,
			}
			cmds = append(cmds, rm.submitRule(rule), updateRulesStatus(rmStatusRefreshing))
		}

		return rm, tea.Batch(cmds...)

	default:
		var cmd tea.Cmd
		rm.table, cmd = rm.table.Update(msg)
		return rm, cmd
	}
}

func (rm *rulesModel) View() string {
	switch rm.status {
	case rmStatusInitializing:
		return "Loading rules..."
	case rmStatusRefreshing:
		return "Refreshing..."
	case rmStatusTable:
		return rm.table.View()
	case rmStatusLoadingData:
		return "Loading form data..."
	case rmStatusEntry:
		return rm.form.View()
	}

	return rm.table.View()
}

func (rm *rulesModel) setModuleDimensions() {
	rm.setTableDimensions(rm.availWidth, rm.availHeight)
	rm.setTableDimensions(rm.availWidth, rm.availHeight)
}

func (rm *rulesModel) setTableDimensions(w, h int) {
	t := &rm.table
	enableW := 8
	boardW := 20
	remainingW := w - enableW - boardW
	recipW := remainingW
	t.SetColumns([]table.Column{
		{Title: "ENABLED", Width: enableW},
		{Title: "BOARD", Width: boardW},
		{Title: "RECIPIENT", Width: recipW},
	})

	t.SetRows(rulesToRows(rm.rules))
	t.SetHeight(h)
}

func (rm *rulesModel) prepareForm() tea.Cmd {
	return func() tea.Msg {
		boards, err := rm.SDKClient.ListBoards()
		if err != nil {
			return sdkErrMsg{err: fmt.Errorf("listing boards: %w", err)}
		}
		sortBoards(boards)

		recips, err := rm.SDKClient.ListRecipients()
		if err != nil {
			return sdkErrMsg{err: fmt.Errorf("listing webex recipients: %w", err)}
		}
		sortRecips(recips)

		return ruleFormDataMsg{
			boards: boards,
			recips: recips,
		}
	}
}

func (rm *rulesModel) submitRule(rule *models.NotifierRule) tea.Cmd {
	return func() tea.Msg {
		_, err := rm.SDKClient.CreateNotifierRule(rule)
		if err != nil {
			return sdkErrMsg{err: fmt.Errorf("posting notifier rule: %w", err)}
		}

		rules, err := rm.SDKClient.ListNotifierRules()
		if err != nil {
			return sdkErrMsg{err: fmt.Errorf("listing rules after create: %w", err)}
		}

		return gotRulesMsg{rules: rules}
	}
}

func (rm *rulesModel) deleteRule(id int) tea.Cmd {
	return func() tea.Msg {
		if err := rm.SDKClient.DeleteNotifierRule(id); err != nil {
			return sdkErrMsg{err: fmt.Errorf("deleting rule: %w", err)}
		}

		rules, err := rm.SDKClient.ListNotifierRules()
		if err != nil {
			return sdkErrMsg{err: fmt.Errorf("listing rules after deletion: %w", err)}
		}

		return gotRulesMsg{rules: rules}
	}
}

func (rm *rulesModel) getRules() tea.Cmd {
	return func() tea.Msg {
		rules, err := rm.SDKClient.ListNotifierRules()
		if err != nil {
			return sdkErr{error: err}
		}

		return gotRulesMsg{rules: rules}
	}
}

func (rm *rulesModel) setFormHeight(h int) {
	rm.form.WithHeight(h)
}

func (rm *rulesModel) setRows() tea.Cmd {
	rm.table.SetRows(rulesToRows(rm.rules))
	rm.table.SetCursor(0)
	return nil
}

func rulesToRows(rules []models.NotifierRuleFull) []table.Row {
	var rows []table.Row
	for _, r := range rules {
		rows = append(rows, []string{boolToIcon(r.Enabled), r.BoardName, r.RecipientName})
	}

	return rows
}

func ruleEntryForm(boards []models.Board, recips []models.WebexRecipient, result *rulesFormResult, height int) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[models.Board]().
				Title("Connectwise Board").
				Options(boardsToFormOpts(boards)...).
				Value(&result.board),
		),
		huh.NewGroup(
			huh.NewSelect[models.WebexRecipient]().
				Title("Webex Recipient").
				Options(recipsToFormOpts(recips)...).
				Value(&result.recip),
		),
	).WithTheme(huh.ThemeBase()).WithHeight(height).WithShowHelp(false)
}

func updateRulesStatus(status rulesModelStatus) tea.Cmd {
	return func() tea.Msg {
		return updateRmStatusMsg{status: status}
	}
}
