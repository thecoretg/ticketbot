package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/thecoretg/ticketbot/internal/models"
)

type (
	syncModel struct {
		parent *Model

		table          table.Model
		form           *huh.Form
		formResult     *syncFormResult
		status         subModelStatus
		previousStatus subModelStatus
		errorMsg       error
	}

	syncFormResult struct {
		syncBoards        bool
		syncWxRecips      bool
		syncTickets       bool
		syncTicketsBoards []models.Board
	}

	syncStartedMsg     struct{}
	syncStatusCheckMsg struct{}
	syncCompleteMsg    struct{}
	gotSyncFormDataMsg struct{ boards []models.Board }
)

func newSyncModel(parent *Model) *syncModel {
	sm := &syncModel{
		parent:     parent,
		table:      newTable(),
		formResult: &syncFormResult{},
		status:     statusMain,
	}
	sm.setModuleDimensions()
	return sm
}

func (sm *syncModel) Init() tea.Cmd {
	return nil
}

func (sm *syncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "enter" && sm.status == statusError:
			sm.errorMsg = nil
			sm.status = sm.previousStatus
			return sm, nil
		case key.Matches(msg, allKeys.newItem) && sm.status == statusMain:
			sm.status = statusLoadingFormData
			return sm, tea.Batch(sm.prepareForm())
		}

	case resizeModelsMsg:
		sm.parent.width = msg.w
		sm.parent.availHeight = msg.h
		sm.setModuleDimensions()
		if sm.status == statusInit {
			sm.status = statusMain
		}

	case gotSyncFormDataMsg:
		sm.formResult = &syncFormResult{}
		sm.form = syncEntryForm(msg.boards, sm.formResult, sm.parent.availHeight)
		sm.status = statusEntry
		return sm, sm.form.Init()

	case syncStartedMsg:
		sm.status = statusSyncing
		return sm, sm.checkSyncStatus()

	case syncStatusCheckMsg:
		return sm, sm.checkSyncStatus()

	case syncCompleteMsg:
		sm.status = statusMain
		return sm, nil

	case errMsg:
		if sm.status == statusLoadingFormData || sm.status == statusSyncing {
			sm.previousStatus = statusMain
		} else {
			sm.previousStatus = sm.status
		}
		sm.errorMsg = msg.error
		sm.status = statusError
	}

	var cmds []tea.Cmd
	switch sm.status {

	case statusEntry:
		form, cmd := sm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			sm.form = f
		}

		cmds = append(cmds, cmd)
		switch sm.form.State {
		case huh.StateAborted:
			sm.status = statusMain

		case huh.StateCompleted:
			res := sm.formResult
			if !res.syncBoards && !res.syncWxRecips && !res.syncTickets {
				sm.status = statusMain
			} else {
				payload := &models.SyncPayload{
					CWBoards:           res.syncBoards,
					WebexRecipients:    res.syncWxRecips,
					CWTickets:          res.syncTickets,
					BoardIDs:           []int{},
					MaxConcurrentSyncs: 5,
				}

				if res.syncTickets && len(res.syncTicketsBoards) > 0 {
					for _, b := range res.syncTicketsBoards {
						payload.BoardIDs = append(payload.BoardIDs, b.ID)
					}
				}

				sm.status = statusSyncing
				cmds = append(cmds, sm.startSync(payload))
			}
		}

	default:
		var cmd tea.Cmd
		sm.table, cmd = sm.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return sm, tea.Batch(cmds...)
}

func (sm *syncModel) View() string {
	switch sm.status {
	case statusInit:
		return fillSpaceCentered(useSpinner(spn, "Initializing sync..."), sm.parent.width, sm.parent.availHeight)
	case statusError:
		return renderErrorView(sm.errorMsg, sm.parent.width, sm.parent.availHeight)
	case statusMain:
		syncInfo := "Sync boards, Webex recipients, or tickets.\n\n" +
			"These are only really necessary when starting a fresh server.\n" +
			"If syncing tickets, expect it to take anywhere from 5 to 30 minutes depending on selected boards. " +
			"For details on progress, see server logs.\n\n" +
			"When the sync is complete, you will be returned to this page.\nPress 'n' to begin."
		return fillSpaceCentered(syncInfo, sm.parent.width, sm.parent.availHeight)
	case statusLoadingFormData:
		return fillSpaceCentered(useSpinner(spn, "Loading sync options..."), sm.parent.width, sm.parent.availHeight)
	case statusEntry:
		return sm.form.View()
	case statusSyncing:
		return fillSpaceCentered(useSpinner(spn, "Syncing...this may take a while..."), sm.parent.width, sm.parent.availHeight)
	}

	return fillSpaceCentered("Press 'n' to start a new sync", sm.parent.width, sm.parent.availHeight)
}

func (sm *syncModel) Status() subModelStatus {
	return sm.status
}

func (sm *syncModel) Form() *huh.Form {
	return sm.form
}

func (sm *syncModel) Table() table.Model {
	return sm.table
}

func (sm *syncModel) setModuleDimensions() {
	w := sm.parent.width
	h := sm.parent.availHeight
	sm.table.SetHeight(h)
	sm.table.SetWidth(w)
}

func (sm *syncModel) prepareForm() tea.Cmd {
	return func() tea.Msg {
		boards, err := sm.parent.SDKClient.ListBoards()
		if err != nil {
			return errMsg{fmt.Errorf("listing boards: %w", err)}
		}

		// Sort boards if any exist
		if len(boards) > 0 {
			sortBoards(boards)
		}

		return gotSyncFormDataMsg{
			boards: boards,
		}
	}
}

func (sm *syncModel) startSync(payload *models.SyncPayload) tea.Cmd {
	return func() tea.Msg {
		if err := sm.parent.SDKClient.Sync(payload); err != nil {
			return errMsg{fmt.Errorf("starting sync: %w", err)}
		}

		return syncStartedMsg{}
	}
}

func (sm *syncModel) checkSyncStatus() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(3 * time.Second)

		syncing, err := sm.parent.SDKClient.GetSyncStatus()
		if err != nil {
			return errMsg{fmt.Errorf("checking sync status: %w", err)}
		}

		if syncing {
			return syncStatusCheckMsg{}
		}

		return syncCompleteMsg{}
	}
}

func syncEntryForm(boards []models.Board, result *syncFormResult, height int) *huh.Form {
	hasBoards := len(boards) > 0
	var boardOpts []huh.Option[models.Board]

	var ticketField huh.Field
	if hasBoards {
		boardOpts = boardsToFormOpts(boards)
		ticketField = confirmationField("Sync Tickets?", &result.syncTickets)
	} else {
		ticketField = huh.NewNote().
			Title("No boards found").
			Description("Run an initial board sync in order to sync tickets.")
	}

	return huh.NewForm(
		huh.NewGroup(
			confirmationField("Sync Boards?", &result.syncBoards),
			confirmationField("Sync Webex Recipients?", &result.syncWxRecips),
			ticketField,
		),
		huh.NewGroup(
			huh.NewMultiSelect[models.Board]().
				Title("Select boards to sync tickets from").
				Description("Leave empty to sync all boards").
				Options(boardOpts...).
				Value(&result.syncTicketsBoards),
		).WithHideFunc(func() bool { return !result.syncTickets }),
	).WithTheme(huh.ThemeBase16()).WithHeight(height + 1).WithShowHelp(false)
}
