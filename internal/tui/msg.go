package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thecoretg/ticketbot/internal/models"
)

type (
	modelType          int
	switchModelMsg     struct{ modelType }
	resizeModelsMsg    struct{ w, h int }
	updateEntryModeMsg struct{ enable bool }
	sdkErrMsg          struct{ err error }

	ruleFormDataMsg struct {
		boards []models.Board
		recips []models.WebexRecipient
	}
)

const (
	modelTypeRules modelType = iota
	modelTypeFwds
)

func switchModel(m modelType) tea.Cmd {
	return func() tea.Msg {
		return switchModelMsg{m}
	}
}

func resizeModels(w, h int) tea.Cmd {
	return func() tea.Msg {
		return resizeModelsMsg{w, h}
	}
}

func updateEntryMode(enable bool) tea.Cmd {
	return func() tea.Msg {
		return updateEntryModeMsg{enable: enable}
	}
}

func (rm *rulesModel) prepareForm() tea.Cmd {
	return func() tea.Msg {
		boards, err := rm.SDKClient.ListBoards()
		if err != nil {
			return sdkErrMsg{err: fmt.Errorf("listing boards: %w", err)}
		}

		recips, err := rm.SDKClient.ListRecipients()
		if err != nil {
			return sdkErrMsg{err: fmt.Errorf("listing webex recipients: %w", err)}
		}

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

func (rm *rulesModel) getRules() tea.Cmd {
	return func() tea.Msg {
		rules, err := rm.SDKClient.ListNotifierRules()
		if err != nil {
			return sdkErr{error: err}
		}

		return gotRulesMsg{rules: rules}
	}
}
