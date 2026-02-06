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

var spn = newSpinner(spinner.Line, green)

type Model struct {
	sdkClient     *sdk.Client
	initialized   bool
	currentUserID int
	currentKeyID  int
	currentAPIKey string
	activeModel   subModel
	rulesModel    *rulesModel
	fwdsModel     *fwdsModel
	usersModel    *usersModel
	apiKeysModel  *apiKeysModel
	syncModel     *syncModel
	help          help.Model
	width         int
	height        int
	availHeight   int
}

type modelsReadyMsg struct {
	rules   *rulesModel
	fwds    *fwdsModel
	users   *usersModel
	apiKeys *apiKeysModel
	sync    *syncModel
}

type subModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	Status() subModelStatus
	Table() table.Model
	Form() *huh.Form
	ModelType() modelType
}

func NewModel(sl *sdk.Client, apiKey string) *Model {
	return &Model{
		sdkClient:     sl,
		currentAPIKey: apiKey,
		help:          help.New(),
	}
}

func (m *Model) createSubModels() tea.Cmd {
	return func() tea.Msg {
		rules, err := m.sdkClient.ListNotifierRules()
		if err != nil {
			return errMsg{fmt.Errorf("listing initial rules: %w", err)}
		}

		fwds, err := m.sdkClient.ListUserForwards(map[string]string{"filter": "active"})
		if err != nil {
			return errMsg{fmt.Errorf("listing initial forwards: %w", err)}
		}

		users, err := m.sdkClient.ListUsers()
		if err != nil {
			return errMsg{fmt.Errorf("listing initial users: %w", err)}
		}

		apiKeys, err := m.sdkClient.ListAPIKeys()
		if err != nil {
			return errMsg{fmt.Errorf("listing initial API keys: %w", err)}
		}

		return modelsReadyMsg{
			rules:   newRulesModel(m, rules),
			fwds:    newFwdsModel(m, fwds),
			users:   newUsersModel(m, users),
			apiKeys: newAPIKeysModel(m, apiKeys, users),
			sync:    newSyncModel(m),
		}
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(spn.Tick, m.getCurrentUser(), m.identifyCurrentKey())
}

func (m *Model) getCurrentUser() tea.Cmd {
	return func() tea.Msg {
		user, err := m.sdkClient.GetCurrentUser()
		if err != nil {
			return errMsg{fmt.Errorf("getting current user: %w", err)}
		}

		return gotCurrentUserMsg{userID: user.ID}
	}
}

func (m *Model) identifyCurrentKey() tea.Cmd {
	return func() tea.Msg {
		if m.currentAPIKey == "" {
			return gotCurrentKeyMsg{keyID: 0}
		}

		keys, err := m.sdkClient.ListAPIKeys()
		if err != nil {
			return errMsg{fmt.Errorf("getting API keys to identify current: %w", err)}
		}

		for _, k := range keys {
			if err := compareBcryptHash(k.KeyHash, m.currentAPIKey); err == nil {
				return gotCurrentKeyMsg{keyID: k.ID}
			}
		}

		return gotCurrentKeyMsg{keyID: 0}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		_, hv := m.helpViewSize()
		th := lipgloss.Height(m.headerView())
		m.availHeight = m.height - hv - th

		if !m.initialized {
			return m, m.createSubModels()
		}

		cmds = append(cmds, resizeModels(m.width, m.availHeight))
	case tea.KeyMsg:
		switch {
		case !isGlobalKey(msg):
			am, cmd := m.activeModel.Update(msg)
			switch am := am.(type) {
			case *rulesModel:
				m.rulesModel = am
			case *fwdsModel:
				m.fwdsModel = am
			case *usersModel:
				m.usersModel = am
			case *apiKeysModel:
				m.apiKeysModel = am
			case *syncModel:
				m.syncModel = am
			}

			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		case key.Matches(msg, allKeys.quit):
			// quit key should exit form if in a form (handled in submodels)
			if m.activeModel.Status() != statusEntry {
				return m, tea.Quit
			}
		case key.Matches(msg, allKeys.switchModelRules):
			return m, switchModel(modelTypeRules)
		case key.Matches(msg, allKeys.switchModelFwds):
			return m, switchModel(modelTypeFwds)
		case key.Matches(msg, allKeys.switchModelUsers):
			return m, switchModel(modelTypeUsers)
		case key.Matches(msg, allKeys.switchModelAPIKeys):
			return m, switchModel(modelTypeAPIKeys)
		case key.Matches(msg, allKeys.switchModelSync):
			return m, switchModel(modelTypeSync)
		case key.Matches(msg, allKeys.nextModel):
			return m, nextModel(m.activeModel.ModelType())
		case key.Matches(msg, allKeys.prevModel):
			return m, prevModel(m.activeModel.ModelType())
		}

	case modelsReadyMsg:
		m.rulesModel = msg.rules
		m.fwdsModel = msg.fwds
		m.usersModel = msg.users
		m.apiKeysModel = msg.apiKeys
		m.syncModel = msg.sync
		m.activeModel = m.rulesModel
		m.initialized = true
		return m, tea.Batch(m.rulesModel.Init(), m.fwdsModel.Init(), m.usersModel.Init(), m.apiKeysModel.Init(), m.syncModel.Init())

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
		case modelTypeUsers:
			if m.activeModel != m.usersModel {
				m.activeModel = m.usersModel
			}
		case modelTypeAPIKeys:
			if m.activeModel != m.apiKeysModel {
				m.activeModel = m.apiKeysModel
			}
		case modelTypeSync:
			if m.activeModel != m.syncModel {
				m.activeModel = m.syncModel
			}
		}

	case gotCurrentUserMsg:
		m.currentUserID = msg.userID
	case gotCurrentKeyMsg:
		m.currentKeyID = msg.keyID
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
	case m.usersModel:
		users, cmd := m.usersModel.Update(msg)
		if u, ok := users.(*usersModel); ok {
			m.usersModel = u
		}
		cmds = append(cmds, cmd)
	case m.apiKeysModel:
		apiKeys, cmd := m.apiKeysModel.Update(msg)
		if ak, ok := apiKeys.(*apiKeysModel); ok {
			m.apiKeysModel = ak
		}
		cmds = append(cmds, cmd)
	case m.syncModel:
		sync, cmd := m.syncModel.Update(msg)
		if s, ok := sync.(*syncModel); ok {
			m.syncModel = s
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
	return lipgloss.JoinVertical(lipgloss.Top, m.headerView(), m.activeModel.View(), m.helpView())
}

func (m *Model) headerView() string {
	mod := menuLabelStyle.Render("CTRL + ")
	r := m.tabStyle(m.rulesModel, "[R] RULES")
	f := m.tabStyle(m.fwdsModel, "[F] FORWARDS")
	u := m.tabStyle(m.usersModel, "[U] USERS")
	k := m.tabStyle(m.apiKeysModel, "[A] KEYS")
	s := m.tabStyle(m.syncModel, "[S] SYNC")

	tabs := []string{r, f, u, k, s}
	sep := " / "
	content := lipgloss.JoinHorizontal(lipgloss.Bottom, mod, strings.Join(tabs, sep), " ")
	avail := m.width - lipgloss.Width(content)
	avail = max(0, avail)
	line := lipgloss.NewStyle().Foreground(green).Render(strings.Repeat(lipgloss.NormalBorder().Bottom, avail))
	return lipgloss.JoinHorizontal(lipgloss.Bottom, content, line)
}

func (m *Model) tabStyle(s subModel, title string) string {
	tab := menuLabelStyle.Render(title)
	if m.activeModel == s {
		tab = activeMenuLabelStyle.Render(title)
	}

	return tab
}
