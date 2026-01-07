package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

var (
	currentErr error
	spn        = newSpinner(spinner.Line, green)
)

type Model struct {
	SDKClient   *sdk.Client
	initialized bool
	activeModel subModel
	rulesModel  *rulesModel
	fwdsModel   *fwdsModel
	help        help.Model
	width       int
	height      int
}

type modelsReadyMsg struct {
	rules *rulesModel
	fwds  *fwdsModel
}

type subModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	Status() subModelStatus
	Table() table.Model
	Form() *huh.Form
}

func NewModel(sl *sdk.Client) *Model {
	return &Model{
		SDKClient: sl,
		help:      help.New(),
	}
}

func (m *Model) createSubModels(w, h int) tea.Cmd {
	return func() tea.Msg {
		rules, err := m.SDKClient.ListNotifierRules()
		if err != nil {
			return errMsg{fmt.Errorf("listing initial rules: %w", err)}
		}

		fwds, err := m.SDKClient.ListUserForwards()
		if err != nil {
			return errMsg{fmt.Errorf("listing initial forwards: %w", err)}
		}

		rp := rulesModelParams{
			sdkClient:     m.SDKClient,
			initialRules:  rules,
			initialWidth:  w,
			initialHeight: h,
		}

		fp := fwdsModelParams{
			sdkClient:     m.SDKClient,
			initialFwds:   fwds,
			initialWidth:  w,
			initialHeight: h,
		}

		return modelsReadyMsg{
			rules: newRulesModel(rp),
			fwds:  newFwdsModel(fp),
		}
	}
}

func (m *Model) Init() tea.Cmd {
	return spn.Tick
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		_, hv := m.helpViewSize()
		ev := lipgloss.Height(errView(currentErr))
		th := lipgloss.Height(m.headerView())

		if !m.initialized {
			return m, m.createSubModels(m.width, m.height-hv-ev-th)
		}

		cmds = append(cmds, resizeModels(m.width, m.height-hv-ev-th))
	case tea.KeyMsg:
		switch {
		case !isGlobalKey(msg):
			am, cmd := m.activeModel.Update(msg)
			switch am := am.(type) {
			case *rulesModel:
				m.rulesModel = am
			case *fwdsModel:
				m.fwdsModel = am
			}

			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		case key.Matches(msg, allKeys.quit):
			if m.activeModel.Status().quittable() {
				return m, tea.Quit
			}
		case key.Matches(msg, allKeys.clearErr):
			if currentErr != nil {
				currentErr = nil
			}
		case key.Matches(msg, allKeys.switchModelRules):
			return m, switchModel(modelTypeRules)
		case key.Matches(msg, allKeys.switchModelFwds):
			return m, switchModel(modelTypeFwds)
		}

	case modelsReadyMsg:
		m.rulesModel = msg.rules
		m.fwdsModel = msg.fwds
		m.activeModel = m.rulesModel
		m.initialized = true
		return m, tea.Batch(m.rulesModel.Init(), m.fwdsModel.Init())

	case switchModelMsg:
		switch msg.modelType {
		case modelTypeRules:
			if m.activeModel != m.rulesModel {
				m.activeModel = m.rulesModel
			}
		case modelTypeFwds:
			if m.activeModel != m.fwdsModel {
				m.activeModel = m.fwdsModel
			}
		}
	case errMsg:
		currentErr = msg.error
	}

	switch m.activeModel {
	case m.rulesModel:
		rules, cmd := m.rulesModel.Update(msg)
		if r, ok := rules.(*rulesModel); ok {
			m.rulesModel = r
		}
		cmds = append(cmds, cmd)
	case m.fwdsModel:
		fwds, cmd := m.fwdsModel.Update(msg)
		if f, ok := fwds.(*fwdsModel); ok {
			m.fwdsModel = f
		}
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	spn, cmd = spn.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if !m.initialized {
		s := useSpinner(spn, "Initializing...")
		return fillSpaceCentered(s, m.width, m.height)
	}
	return lipgloss.JoinVertical(lipgloss.Top, m.headerView(), m.activeModel.View(), errView(currentErr), m.helpView())
}

func (m *Model) headerView() string {
	rl := "[CTRL+R] RULES"
	fl := "[CTRL+F] FORWARDS "
	rulesTab := menuLabelStyle.Render(rl)
	if m.activeModel == m.rulesModel {
		rulesTab = activeMenuLabelStyle.Render(rl)
	}

	fwdsTab := menuLabelStyle.Render(fl)
	if m.activeModel == m.fwdsModel {
		fwdsTab = activeMenuLabelStyle.Render(fl)
	}

	avail := m.width - lipgloss.Width(rulesTab) - lipgloss.Width(fwdsTab)
	avail = max(0, avail)
	line := lipgloss.NewStyle().Foreground(green).Render(strings.Repeat(lipgloss.NormalBorder().Bottom, avail))
	tabs := lipgloss.JoinHorizontal(lipgloss.Bottom, rulesTab, " / ", fwdsTab, line)
	return tabs
}

func errView(err error) string {
	if err != nil {
		return errStyle.Render("Error: " + err.Error())
	}
	return ""
}
