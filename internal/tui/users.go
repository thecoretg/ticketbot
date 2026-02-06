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
	usersModel struct {
		parent *Model

		usersLoaded       bool
		table             table.Model
		form              *huh.Form
		formResult        *usersFormResult
		status            subModelStatus
		previousStatus    subModelStatus
		users             []models.APIUser
		userToDelete      models.APIUser
		userDeleteConfirm bool
		errorMsg          error
	}

	usersFormResult struct {
		email string
	}

	refreshUsersMsg   struct{}
	gotUsersMsg       struct{ users []models.APIUser }
	gotCurrentUserMsg struct{ userID int }
	gotCurrentKeyMsg  struct{ keyID int }
)

func newUsersModel(parent *Model, initialUsers []models.APIUser) *usersModel {
	um := &usersModel{
		parent:     parent,
		users:      initialUsers,
		table:      newTable(),
		formResult: &usersFormResult{},
		status:     statusMain,
	}
	um.setModuleDimensions()
	return um
}

func (um *usersModel) ModelType() modelType {
	return modelTypeUsers
}

func (um *usersModel) Init() tea.Cmd {
	return nil
}

func (um *usersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "enter" && um.status == statusError:
			um.errorMsg = nil
			um.status = um.previousStatus
			return um, nil
		case key.Matches(msg, allKeys.newItem) && um.status == statusMain:
			um.formResult = &usersFormResult{}
			um.form = userEntryForm(um.formResult, um.parent.availHeight)
			um.status = statusEntry
			return um, um.form.Init()
		case key.Matches(msg, allKeys.deleteItem) && um.status == statusMain:
			if len(um.users) > 0 {
				um.userToDelete = um.users[um.table.Cursor()]
				if um.parent.currentUserID > 0 && um.userToDelete.ID == um.parent.currentUserID {
					return um, func() tea.Msg {
						return errMsg{fmt.Errorf("cannot delete your own user account (ID: %d)", um.parent.currentUserID)}
					}
				}
				um.form = confirmationForm(fmt.Sprintf("Delete user %s?", um.userToDelete.EmailAddress), &um.userDeleteConfirm, um.parent.availHeight)
				um.status = statusConfirm
				return um, um.form.Init()
			}
		}

	case resizeModelsMsg:
		um.setModuleDimensions()
		if um.status == statusInit {
			um.status = statusMain
		}

	case refreshUsersMsg:
		return um, um.getUsers()

	case gotUsersMsg:
		um.users = msg.users
		um.usersLoaded = true
		um.status = statusMain
		return um, tea.Batch(um.setRows())

	case confirmDeleteMsg:
		var id int
		if um.userDeleteConfirm {
			id = um.userToDelete.ID
		}

		// reset values
		um.userDeleteConfirm = false
		um.userToDelete = models.APIUser{}

		if id != 0 {
			return um, um.deleteUser(id)
		}
		um.status = statusMain

	case errMsg:
		// If we're in a transient/loading status, go back to main after error
		if um.status == statusRefresh {
			um.previousStatus = statusMain
		} else {
			um.previousStatus = um.status
		}
		um.errorMsg = msg.error
		um.status = statusError
	}

	var cmds []tea.Cmd
	switch um.status {

	case statusEntry, statusConfirm:
		form, cmd := um.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			um.form = f
		}

		cmds = append(cmds, cmd)
		switch um.form.State {
		case huh.StateAborted:
			um.status = statusMain

		case huh.StateCompleted:
			switch um.status {
			case statusConfirm:
				um.status = statusRefresh
				cmds = append(cmds, completeConfirmForm())
			case statusEntry:
				res := um.formResult
				um.status = statusRefresh
				cmds = append(cmds, um.submitUser(res.email))
			}
		}

	default:
		var cmd tea.Cmd
		um.table, cmd = um.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return um, tea.Batch(cmds...)
}

func (um *usersModel) View() string {
	switch um.status {
	case statusInit:
		return fillSpaceCentered(useSpinner(spn, "Loading users..."), um.parent.width, um.parent.availHeight)
	case statusRefresh:
		return fillSpaceCentered(useSpinner(spn, "Refreshing..."), um.parent.width, um.parent.availHeight)
	case statusError:
		return renderErrorView(um.errorMsg, um.parent.width, um.parent.availHeight)
	case statusMain:
		return um.table.View()
	case statusEntry, statusConfirm:
		return um.form.View()
	}

	return um.table.View()
}

func (um *usersModel) Status() subModelStatus {
	return um.status
}

func (um *usersModel) Form() *huh.Form {
	return um.form
}

func (um *usersModel) Table() table.Model {
	return um.table
}

func (um *usersModel) setModuleDimensions() {
	um.setTableDimensions()
}

func (um *usersModel) setTableDimensions() {
	w := um.parent.width
	h := um.parent.availHeight
	t := &um.table
	idW := 6
	emailW := 40
	remainingW := w - idW - emailW
	createdW := remainingW
	t.SetColumns([]table.Column{
		{Title: "ID", Width: idW},
		{Title: "EMAIL", Width: emailW},
		{Title: "CREATED", Width: createdW},
	})

	t.SetRows(usersToRows(um.users))
	t.SetHeight(h)
}

func (um *usersModel) submitUser(email string) tea.Cmd {
	return func() tea.Msg {
		_, err := um.parent.sdkClient.CreateUser(email)
		if err != nil {
			return errMsg{fmt.Errorf("creating user: %w", err)}
		}

		return refreshUsersMsg{}
	}
}

func (um *usersModel) deleteUser(id int) tea.Cmd {
	return func() tea.Msg {
		if err := um.parent.sdkClient.DeleteUser(id); err != nil {
			return errMsg{fmt.Errorf("deleting user: %w", err)}
		}

		return refreshUsersMsg{}
	}
}

func (um *usersModel) getUsers() tea.Cmd {
	return func() tea.Msg {
		users, err := um.parent.sdkClient.ListUsers()
		if err != nil {
			return errMsg{fmt.Errorf("getting users: %w", err)}
		}

		return gotUsersMsg{users: users}
	}
}

func (um *usersModel) setRows() tea.Cmd {
	um.table.SetRows(usersToRows(um.users))
	um.table.SetCursor(0)
	return nil
}

func usersToRows(users []models.APIUser) []table.Row {
	if len(users) == 0 {
		return []table.Row{
			{
				"NO", "USERS", "FOUND",
			},
		}
	}
	var rows []table.Row
	for _, u := range users {
		rows = append(rows, []string{
			fmt.Sprintf("%d", u.ID),
			u.EmailAddress,
			u.CreatedOn.Format("2006-01-02 15:04"),
		})
	}

	return rows
}

func userEntryForm(result *usersFormResult, height int) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Email Address").
				Value(&result.email).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("email address is required")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeBase16()).WithHeight(height + 1).WithShowHelp(false)
}
