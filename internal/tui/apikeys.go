package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/thecoretg/ticketbot/internal/models"
)

type (
	apiKeysModel struct {
		parent *Model

		keysLoaded       bool
		table            table.Model
		form             *huh.Form
		formResult       *apiKeysFormResult
		status           subModelStatus
		previousStatus   subModelStatus
		keys             []models.APIKey
		keyToDelete      models.APIKey
		users            []models.APIUser
		keyDeleteConfirm bool
		createdKey       string
		errorMsg         error
	}

	apiKeysFormDataMsg struct {
		users []models.APIUser
	}

	apiKeysFormResult struct {
		user models.APIUser
	}

	refreshAPIKeysMsg struct{}
	gotAPIKeysMsg     struct{ keys []models.APIKey }
	showCreatedKeyMsg struct{}
)

func newAPIKeysModel(parent *Model, initialKeys []models.APIKey, initialUsers []models.APIUser) *apiKeysModel {
	akm := &apiKeysModel{
		parent:     parent,
		keys:       initialKeys,
		users:      initialUsers,
		table:      newTable(),
		formResult: &apiKeysFormResult{},
		status:     statusMain,
	}
	akm.setModuleDimensions()
	return akm
}

func (akm *apiKeysModel) Init() tea.Cmd {
	return nil
}

func (akm *apiKeysModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "enter" && akm.status == statusError:
			akm.errorMsg = nil
			akm.status = akm.previousStatus
			return akm, nil
		case key.Matches(msg, allKeys.newItem) && akm.status == statusMain:
			akm.status = statusLoadingFormData
			return akm, akm.prepareForm()
		case key.Matches(msg, allKeys.deleteItem) && akm.status == statusMain:
			if len(akm.keys) > 0 {
				akm.keyToDelete = akm.keys[akm.table.Cursor()]
				if akm.parent.currentKeyID > 0 && akm.keyToDelete.ID == akm.parent.currentKeyID {
					return akm, func() tea.Msg {
						return errMsg{fmt.Errorf("cannot delete the currently used API key (ID: %d)", akm.parent.currentKeyID)}
					}
				}
				akm.form = confirmationForm(fmt.Sprintf("Delete API key ID %d?", akm.keyToDelete.ID), &akm.keyDeleteConfirm, akm.parent.availHeight)
				akm.status = statusConfirm
				return akm, akm.form.Init()
			}
		case msg.String() == "enter" && akm.status == statusShowKey:
			akm.createdKey = ""
			akm.status = statusRefresh
			return akm, func() tea.Msg { return refreshAPIKeysMsg{} }
		}

	case resizeModelsMsg:
		akm.setModuleDimensions()
		if akm.status == statusInit {
			akm.status = statusMain
		}

	case refreshAPIKeysMsg:
		return akm, akm.getKeys()

	case gotAPIKeysMsg:
		akm.keys = msg.keys
		akm.keysLoaded = true
		akm.status = statusMain
		return akm, tea.Batch(akm.setRows())

	case apiKeysFormDataMsg:
		akm.formResult = &apiKeysFormResult{}
		akm.form = apiKeyEntryForm(msg.users, akm.formResult, akm.parent.availHeight)
		akm.status = statusEntry
		return akm, akm.form.Init()

	case confirmDeleteMsg:
		var id int
		if akm.keyDeleteConfirm {
			id = akm.keyToDelete.ID
		}

		// reset values
		akm.keyDeleteConfirm = false
		akm.keyToDelete = models.APIKey{}

		if id != 0 {
			return akm, akm.deleteKey(id)
		}
		akm.status = statusMain

	case showCreatedKeyMsg:
		akm.status = statusShowKey
		return akm, nil

	case errMsg:
		// If we're in a transient/loading status, go back to main after error
		if akm.status == statusLoadingFormData || akm.status == statusRefresh {
			akm.previousStatus = statusMain
		} else {
			akm.previousStatus = akm.status
		}
		akm.errorMsg = msg.error
		akm.status = statusError
	}

	var cmds []tea.Cmd
	switch akm.status {

	case statusEntry, statusConfirm:
		form, cmd := akm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			akm.form = f
		}

		cmds = append(cmds, cmd)
		switch akm.form.State {
		case huh.StateAborted:
			akm.status = statusMain

		case huh.StateCompleted:
			switch akm.status {
			case statusConfirm:
				akm.status = statusRefresh
				cmds = append(cmds, completeConfirmForm())
			case statusEntry:
				res := akm.formResult
				akm.status = statusRefresh
				cmds = append(cmds, akm.submitKey(res.user.EmailAddress))
			}
		}

	default:
		var cmd tea.Cmd
		akm.table, cmd = akm.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return akm, tea.Batch(cmds...)
}

func (akm *apiKeysModel) View() string {
	switch akm.status {
	case statusInit:
		return fillSpaceCentered(useSpinner(spn, "Loading API keys..."), akm.parent.width, akm.parent.availHeight)
	case statusRefresh:
		return fillSpaceCentered(useSpinner(spn, "Refreshing..."), akm.parent.width, akm.parent.availHeight)
	case statusLoadingFormData:
		return fillSpaceCentered(useSpinner(spn, "Loading users..."), akm.parent.width, akm.parent.availHeight)
	case statusShowKey:
		return akm.showKeyView()
	case statusError:
		return renderErrorView(akm.errorMsg, akm.parent.width, akm.parent.availHeight)
	case statusMain:
		return akm.table.View()
	case statusEntry, statusConfirm:
		return akm.form.View()
	}

	return akm.table.View()
}

func (akm *apiKeysModel) showKeyView() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  API Key Created Successfully!\n\n")
	b.WriteString("  IMPORTANT: Copy this key now. It will not be shown again.\n\n")
	fmt.Fprintf(&b, "  %s\n\n", akm.createdKey)
	b.WriteString("  Press ENTER to continue...")
	return b.String()
}

func (akm *apiKeysModel) Status() subModelStatus {
	return akm.status
}

func (akm *apiKeysModel) Form() *huh.Form {
	return akm.form
}

func (akm *apiKeysModel) Table() table.Model {
	return akm.table
}

func (akm *apiKeysModel) setModuleDimensions() {
	akm.setTableDimensions()
}

func (akm *apiKeysModel) setTableDimensions() {
	w := akm.parent.width
	h := akm.parent.availHeight
	t := &akm.table
	idW := 6
	userIDW := 8
	emailW := 20
	hintW := 10
	remainingW := w - idW - userIDW - emailW - hintW
	createdW := remainingW
	t.SetColumns([]table.Column{
		{Title: "ID", Width: idW},
		{Title: "USER_ID", Width: userIDW},
		{Title: "EMAIL", Width: emailW},
		{Title: "HINT", Width: hintW},
		{Title: "CREATED", Width: createdW},
	})

	t.SetRows(apiKeysToRows(akm.keys, akm.users))
	t.SetHeight(h)
}

func (akm *apiKeysModel) prepareForm() tea.Cmd {
	return func() tea.Msg {
		users, err := akm.parent.SDKClient.ListUsers()
		if err != nil {
			return errMsg{fmt.Errorf("loading users: %w", err)}
		}

		if len(users) == 0 {
			return errMsg{fmt.Errorf("no users available")}
		}

		return apiKeysFormDataMsg{users: users}
	}
}

func (akm *apiKeysModel) submitKey(email string) tea.Cmd {
	return func() tea.Msg {
		key, err := akm.parent.SDKClient.CreateAPIKey(email)
		if err != nil {
			return errMsg{fmt.Errorf("creating API key: %w", err)}
		}

		akm.createdKey = key
		return showCreatedKeyMsg{}
	}
}

func (akm *apiKeysModel) deleteKey(id int) tea.Cmd {
	return func() tea.Msg {
		if err := akm.parent.SDKClient.DeleteAPIKey(id); err != nil {
			return errMsg{fmt.Errorf("deleting API key: %w", err)}
		}

		return refreshAPIKeysMsg{}
	}
}

func (akm *apiKeysModel) getKeys() tea.Cmd {
	return func() tea.Msg {
		keys, err := akm.parent.SDKClient.ListAPIKeys()
		if err != nil {
			return errMsg{fmt.Errorf("getting API keys: %w", err)}
		}

		return gotAPIKeysMsg{keys: keys}
	}
}

func (akm *apiKeysModel) setRows() tea.Cmd {
	akm.table.SetRows(apiKeysToRows(akm.keys, akm.users))
	akm.table.SetCursor(0)
	return nil
}

func apiKeysToRows(keys []models.APIKey, users []models.APIUser) []table.Row {
	if len(keys) == 0 {
		return []table.Row{
			{
				"NO", "API", "KEYS", "FOUND",
			},
		}
	}

	userLookup := make(map[int]models.APIUser, len(users))
	for _, u := range users {
		userLookup[u.ID] = u
	}

	var rows []table.Row
	for _, k := range keys {
		hint := "N/A"
		if k.KeyHint != nil && *k.KeyHint != "" {
			hint = "****" + *k.KeyHint
		}

		email := "N/A"
		if u, ok := userLookup[k.UserID]; ok {
			email = u.EmailAddress
		}

		rows = append(rows, []string{
			fmt.Sprintf("%d", k.ID),
			fmt.Sprintf("%d", k.UserID),
			email,
			hint,
			k.CreatedOn.Format("2006-01-02 15:04"),
		})
	}

	return rows
}

func apiKeyEntryForm(users []models.APIUser, result *apiKeysFormResult, height int) *huh.Form {
	opts := usersToFormOpts(users)
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[models.APIUser]().
				Title("Select User").
				Options(opts...).
				Value(&result.user),
		),
	).WithTheme(huh.ThemeBase16()).WithHeight(height + 1).WithShowHelp(false)
}

func usersToFormOpts(users []models.APIUser) []huh.Option[models.APIUser] {
	var opts []huh.Option[models.APIUser]
	for _, u := range users {
		o := huh.NewOption(u.EmailAddress, u)
		opts = append(opts, o)
	}

	return opts
}
