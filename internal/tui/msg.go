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
	noBoardsAvailMsg = errMsg{errors.New("form exited: no boards found")}
	noRecipsAvailMsg = errMsg{errors.New("form exited: no recipients found")}
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
