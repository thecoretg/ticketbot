package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/thecoretg/ticketbot/internal/models"
)

type (
	rulesModel struct {
		parent *Model

		rulesLoaded       bool
		table             table.Model
		form              *huh.Form
		formResult        *rulesFormResult
		status            subModelStatus
		previousStatus    subModelStatus
		rules             []models.NotifierRuleFull
		ruleToDelete      models.NotifierRuleFull
		ruleDeleteConfirm bool
		errorMsg          error
	}

	ruleFormDataMsg struct {
		boards []models.Board
		recips []models.WebexRecipient
	}

	rulesFormResult struct {
		board models.Board
		recip models.WebexRecipient
	}

	refreshRulesMsg struct{}
	gotRulesMsg     struct{ rules []models.NotifierRuleFull }
)

func newRulesModel(parent *Model, initialRules []models.NotifierRuleFull) *rulesModel {
	rm := &rulesModel{
		parent:     parent,
		rules:      initialRules,
		table:      newTable(),
		formResult: &rulesFormResult{},
		status:     statusMain,
	}
	rm.setModuleDimensions()
	return rm
}

func (rm *rulesModel) ModelType() modelType {
	return modelTypeRules
}

func (rm *rulesModel) Init() tea.Cmd {
	return nil
}

func (rm *rulesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "enter" && rm.status == statusError:
			rm.errorMsg = nil
			rm.status = rm.previousStatus
			return rm, nil
		case key.Matches(msg, allKeys.newItem) && rm.status == statusMain:
			rm.status = statusLoadingFormData
			return rm, tea.Batch(rm.prepareForm())
		case key.Matches(msg, allKeys.deleteItem) && rm.status == statusMain:
			if len(rm.rules) > 0 {
				rm.ruleToDelete = rm.rules[rm.table.Cursor()]
				rm.form = confirmationForm("Delete rule?", &rm.ruleDeleteConfirm, rm.parent.availHeight)
				rm.status = statusConfirm
				return rm, rm.form.Init()
			}
		}

	case resizeModelsMsg:
		rm.parent.width = msg.w
		rm.parent.availHeight = msg.h
		rm.setModuleDimensions()
		if rm.status == statusInit {
			rm.status = statusMain
		}

	case refreshRulesMsg:
		return rm, rm.getRules()

	case gotRulesMsg:
		rm.rules = msg.rules
		rm.rulesLoaded = true
		rm.status = statusMain
		return rm, tea.Batch(rm.setRows())

	case ruleFormDataMsg:
		rm.formResult = &rulesFormResult{}
		rm.form = ruleEntryForm(msg.boards, msg.recips, rm.formResult, rm.parent.availHeight)
		rm.status = statusEntry
		return rm, rm.form.Init()

	case confirmDeleteMsg:
		var id int
		if rm.ruleDeleteConfirm {
			id = rm.ruleToDelete.ID
		}

		// reset values
		rm.ruleDeleteConfirm = false
		rm.ruleToDelete = models.NotifierRuleFull{}

		if id != 0 {
			return rm, rm.deleteRule(id)
		}
		rm.status = statusMain

	case errMsg:
		// If we're in a transient/loading status, go back to main after error
		if rm.status == statusLoadingFormData || rm.status == statusRefresh {
			rm.previousStatus = statusMain
		} else {
			rm.previousStatus = rm.status
		}
		rm.errorMsg = msg.error
		rm.status = statusError
	}

	var cmds []tea.Cmd
	switch rm.status {

	case statusEntry, statusConfirm:
		form, cmd := rm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			rm.form = f
		}

		cmds = append(cmds, cmd)
		switch rm.form.State {
		case huh.StateAborted:
			rm.status = statusMain

		case huh.StateCompleted:
			switch rm.status {
			case statusConfirm:
				rm.status = statusRefresh
				cmds = append(cmds, completeConfirmForm())
			case statusEntry:
				res := rm.formResult
				rule := &models.NotifierRule{
					CwBoardID:        res.board.ID,
					WebexRecipientID: res.recip.ID,
					NotifyEnabled:    true,
				}
				rm.status = statusRefresh
				cmds = append(cmds, rm.submitRule(rule))
			}
		}

	default:
		var cmd tea.Cmd
		rm.table, cmd = rm.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return rm, tea.Batch(cmds...)
}

func (rm *rulesModel) View() string {
	switch rm.status {
	case statusInit:
		return fillSpaceCentered(useSpinner(spn, "Loading rules..."), rm.parent.width, rm.parent.availHeight)
	case statusRefresh:
		return fillSpaceCentered(useSpinner(spn, "Refreshing..."), rm.parent.width, rm.parent.availHeight)
	case statusError:
		return renderErrorView(rm.errorMsg, rm.parent.width, rm.parent.availHeight)
	case statusMain:
		return rm.table.View()
	case statusLoadingFormData:
		return fillSpaceCentered(useSpinner(spn, "Loading form data..."), rm.parent.width, rm.parent.availHeight)
	case statusEntry, statusConfirm:
		return rm.form.View()
	}

	return rm.table.View()
}

func (rm *rulesModel) Status() subModelStatus {
	return rm.status
}

func (rm *rulesModel) Form() *huh.Form {
	return rm.form
}

func (rm *rulesModel) Table() table.Model {
	return rm.table
}

func (rm *rulesModel) setModuleDimensions() {
	rm.setTableDimensions()
}

func (rm *rulesModel) setTableDimensions() {
	w := rm.parent.width
	h := rm.parent.availHeight
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
		boards, err := rm.parent.sdkClient.ListBoards()
		if err != nil {
			return errMsg{fmt.Errorf("listing boards: %w", err)}
		}

		if len(boards) == 0 {
			return noBoardsAvailMsg
		}
		sortBoards(boards)

		recips, err := rm.parent.sdkClient.ListRecipients()
		if err != nil {
			return errMsg{fmt.Errorf("listing webex recipients: %w", err)}
		}

		if len(recips) == 0 {
			return noRecipsAvailMsg
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
		_, err := rm.parent.sdkClient.CreateNotifierRule(rule)
		if err != nil {
			return errMsg{fmt.Errorf("creating notifier rule: %w", err)}
		}

		return refreshRulesMsg{}
	}
}

func (rm *rulesModel) deleteRule(id int) tea.Cmd {
	return func() tea.Msg {
		if err := rm.parent.sdkClient.DeleteNotifierRule(id); err != nil {
			return errMsg{fmt.Errorf("deleting notifier rule: %w", err)}
		}

		return refreshRulesMsg{}
	}
}

func (rm *rulesModel) getRules() tea.Cmd {
	return func() tea.Msg {
		rules, err := rm.parent.sdkClient.ListNotifierRules()
		if err != nil {
			return errMsg{fmt.Errorf("getting rules: %w", err)}
		}

		return gotRulesMsg{rules: rules}
	}
}

func (rm *rulesModel) setRows() tea.Cmd {
	sortRules(rm.rules)
	rm.table.SetRows(rulesToRows(rm.rules))
	rm.table.SetCursor(0)
	return nil
}

func rulesToRows(rules []models.NotifierRuleFull) []table.Row {
	if len(rules) == 0 {
		return []table.Row{
			{
				"NO", "RULES", "FOUND",
			},
		}
	}
	var rows []table.Row
	for _, r := range rules {
		recip := fmt.Sprintf("%s (%s)", r.RecipientName, r.RecipientType)
		rows = append(rows, []string{boolToIcon(r.Enabled), r.BoardName, recip})
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
				Options(recipsToFormOpts(recips, nil)...).
				Value(&result.recip),
		),
	).WithTheme(huh.ThemeBase16()).WithHeight(height + 1).WithShowHelp(false) // add +1 to height to account for not showing help
}
