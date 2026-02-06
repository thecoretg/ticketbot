package tui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

type (
	modelType       int
	switchModelMsg  struct{ modelType }
	resizeModelsMsg struct{ w, h int }
	errMsg          struct{ error }
)

var (
	noBoardsAvailMsg = errMsg{errors.New("no boards found")}
	noRecipsAvailMsg = errMsg{errors.New("no recipients found")}
)

const (
	modelTypeRules modelType = iota
	modelTypeFwds
	modelTypeUsers
	modelTypeAPIKeys
	modelTypeSync
)

func nextModel(current modelType) tea.Cmd {
	return func() tea.Msg {
		if current == modelTypeSync {
			return switchModelMsg{modelTypeRules}
		}
		return switchModelMsg{current + 1}
	}
}

func prevModel(current modelType) tea.Cmd {
	return func() tea.Msg {
		if current == modelTypeRules {
			return switchModelMsg{modelTypeSync}
		}
		return switchModelMsg{current - 1}
	}
}

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
