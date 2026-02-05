package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/thecoretg/ticketbot/internal/models"
)

type (
	fwdsModel struct {
		parent *Model

		fwdsLoaded       bool
		table            table.Model
		form             *huh.Form
		formResult       *fwdsFormResult
		status           subModelStatus
		previousStatus   subModelStatus
		fwds             []models.NotifierForwardFull
		fwdToDelete      models.NotifierForwardFull
		fwdDeleteConfirm bool
		errorMsg         error
	}

	fwdsFormDataMsg struct {
		recips []models.WebexRecipient
	}

	fwdsFormResult struct {
		src       models.WebexRecipient
		dst       models.WebexRecipient
		start     string
		end       string
		userKeeps bool
	}

	refreshFwdsMsg struct{}
	gotFwdsMsg     struct{ fwds []models.NotifierForwardFull }
)

var (
	errInvalidDateInput    = errors.New("valid date in format YYYY-MM-DD required")
	errEndEarlierThanStart = errors.New("end time cannot be before start time")
)

func newFwdsModel(parent *Model, initialFwds []models.NotifierForwardFull) *fwdsModel {
	fm := &fwdsModel{
		parent:     parent,
		fwds:       initialFwds,
		table:      newTable(),
		formResult: &fwdsFormResult{},
		status:     statusMain,
	}

	fm.setModuleDimensions()
	return fm
}

func (fm *fwdsModel) Init() tea.Cmd {
	return nil
}

func (fm *fwdsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "enter" && fm.status == statusError:
			fm.errorMsg = nil
			fm.status = fm.previousStatus
			return fm, nil
		case key.Matches(msg, allKeys.newItem) && fm.status == statusMain:
			fm.status = statusLoadingFormData
			return fm, fm.prepareForm()
		case key.Matches(msg, allKeys.deleteItem) && fm.status == statusMain:
			if len(fm.fwds) > 0 {
				fm.fwdToDelete = fm.fwds[fm.table.Cursor()]
				fm.form = confirmationForm("Delete forward?", &fm.fwdDeleteConfirm, fm.parent.availHeight)
				fm.status = statusConfirm
				return fm, fm.form.Init()
			}
		}
	case resizeModelsMsg:
		fm.parent.width = msg.w
		fm.parent.availHeight = msg.h
		fm.setModuleDimensions()
		if fm.status == statusInit {
			fm.status = statusMain
		}

	case refreshFwdsMsg:
		return fm, fm.getFwds()

	case gotFwdsMsg:
		fm.fwds = msg.fwds
		fm.fwdsLoaded = true
		fm.status = statusMain
		return fm, fm.setRows()

	case fwdsFormDataMsg:
		fm.formResult = &fwdsFormResult{}
		fm.form = fwdEntryForm(msg.recips, fm.formResult)
		fm.status = statusEntry
		return fm, fm.form.Init()

	case confirmDeleteMsg:
		var id int
		if fm.fwdDeleteConfirm {
			id = fm.fwdToDelete.ID
		}

		// reset values
		fm.fwdDeleteConfirm = false
		fm.fwdToDelete = models.NotifierForwardFull{}

		if id != 0 {
			return fm, fm.deleteFwd(id)
		}
		fm.status = statusMain

	case errMsg:
		// If we're in a transient/loading status, go back to main after error
		if fm.status == statusLoadingFormData || fm.status == statusRefresh {
			fm.previousStatus = statusMain
		} else {
			fm.previousStatus = fm.status
		}
		fm.errorMsg = msg.error
		fm.status = statusError
	}

	var cmds []tea.Cmd
	switch fm.status {
	case statusEntry, statusConfirm:
		fm.setFormHeight(fm.parent.availHeight)
		form, cmd := fm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			fm.form = f
		}

		cmds = append(cmds, cmd)
		switch fm.form.State {
		case huh.StateAborted:
			fm.status = statusMain

		case huh.StateCompleted:
			switch fm.status {
			case statusConfirm:
				fm.status = statusRefresh
				cmds = append(cmds, completeConfirmForm())
			case statusEntry:
				res := fm.formResult
				fwd := fwdFormResToForm(res)
				fm.status = statusRefresh
				cmds = append(cmds, fm.submitFwd(fwd))
			}
		}

	default:
		var cmd tea.Cmd
		fm.table, cmd = fm.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return fm, tea.Batch(cmds...)
}

func (fm *fwdsModel) View() string {
	switch fm.status {
	case statusInit:
		return fillSpaceCentered(useSpinner(spn, "Loading forwards..."), fm.parent.width, fm.parent.availHeight)
	case statusRefresh:
		return fillSpaceCentered(useSpinner(spn, "Refreshing..."), fm.parent.width, fm.parent.availHeight)
	case statusError:
		return renderErrorView(fm.errorMsg, fm.parent.width, fm.parent.availHeight)
	case statusMain:
		return fm.table.View()
	case statusLoadingFormData:
		return fillSpaceCentered(useSpinner(spn, "Loading form data..."), fm.parent.width, fm.parent.availHeight)
	case statusEntry, statusConfirm:
		return fm.form.View()
	}

	return fm.table.View()
}

func (fm *fwdsModel) Status() subModelStatus {
	return fm.status
}

func (fm *fwdsModel) Form() *huh.Form {
	return fm.form
}

func (fm *fwdsModel) Table() table.Model {
	return fm.table
}

func (fm *fwdsModel) setModuleDimensions() {
	fm.setTableDimensions()
	if fm.form != nil {
		fm.setFormHeight(fm.parent.availHeight)
	}
}

func (fm *fwdsModel) setTableDimensions() {
	w := fm.parent.width
	h := fm.parent.availHeight
	t := &fm.table
	enableW := 8
	keepW := 8
	datesW := 13
	srcW := 25
	remainingW := w - enableW - datesW - keepW - srcW
	destW := remainingW
	t.SetColumns(
		[]table.Column{
			{Title: "ENABLED", Width: enableW},
			{Title: "KEEP", Width: keepW},
			{Title: "DATES", Width: datesW},
			{Title: "SOURCE", Width: srcW},
			{Title: "DESTINATION", Width: destW},
		},
	)
	t.SetRows(fwdsToRows(fm.fwds))
	t.SetHeight(h)
}

func (fm *fwdsModel) setFormHeight(h int) {
	e := fm.form.Errors()
	// start with +1 since we return help view in the main model, not in the form itself
	newH := h - len(e)
	fm.form.WithHeight(newH)
}

func (fm *fwdsModel) prepareForm() tea.Cmd {
	return func() tea.Msg {
		recips, err := fm.parent.SDKClient.ListRecipients()
		if err != nil {
			return errMsg{fmt.Errorf("listing recipients: %w", err)}
		}

		if len(recips) == 0 {
			return noRecipsAvailMsg
		}

		sortRecips(recips)

		return fwdsFormDataMsg{
			recips: recips,
		}
	}
}

func (fm *fwdsModel) submitFwd(fwd *models.NotifierForward) tea.Cmd {
	return func() tea.Msg {
		_, err := fm.parent.SDKClient.CreateUserForward(fwd)
		if err != nil {
			return errMsg{fmt.Errorf("creating notifier forward: %w", err)}
		}

		return refreshFwdsMsg{}
	}
}

func (fm *fwdsModel) deleteFwd(id int) tea.Cmd {
	return func() tea.Msg {
		if err := fm.parent.SDKClient.DeleteUserForward(id); err != nil {
			return errMsg{fmt.Errorf("deleting notifier forward: %w", err)}
		}

		return refreshFwdsMsg{}
	}
}

func (fm *fwdsModel) getFwds() tea.Cmd {
	return func() tea.Msg {
		p := map[string]string{"filter": "active"}
		fwds, err := fm.parent.SDKClient.ListUserForwards(p)
		if err != nil {
			return errMsg{fmt.Errorf("listing notifier forwards: %w", err)}
		}

		return gotFwdsMsg{fwds: fwds}
	}
}

func (fm *fwdsModel) setRows() tea.Cmd {
	fm.table.SetRows(fwdsToRows(fm.fwds))
	fm.table.SetCursor(0)
	return nil
}

func fwdsToRows(fwds []models.NotifierForwardFull) []table.Row {
	if len(fwds) == 0 {
		return []table.Row{
			{
				"NO", "FWDS", "FOUND", "", "",
			},
		}
	}
	var rows []table.Row
	for _, f := range fwds {
		src := fmt.Sprintf("%s (%s)", f.SourceName, shortenSourceType(f.SourceType))
		dst := fmt.Sprintf("%s (%s)", f.DestinationName, shortenSourceType(f.DestinationType))
		sd := "N/A"
		ed := "N/A"
		if f.StartDate != nil {
			sd = f.StartDate.Format("01-02")
		}

		if f.EndDate != nil {
			ed = f.EndDate.Format("01-02")
		}
		dr := fmt.Sprintf("%s - %s", sd, ed)

		rows = append(rows, []string{
			boolToIcon(f.Enabled),
			boolToIcon(f.UserKeepsCopy),
			dr,
			src,
			dst,
		})
	}

	return rows
}

func fwdEntryForm(recips []models.WebexRecipient, result *fwdsFormResult) *huh.Form {
	theme := huh.ThemeBase16()
	theme.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(red)
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[models.WebexRecipient]().
				Title("Source Recipient").
				Options(recipsToFormOpts(recips, nil)...).
				Value(&result.src),
		),
		huh.NewGroup(
			huh.NewSelect[models.WebexRecipient]().
				Title("Destination Recipient").
				Options(recipsToFormOpts(recips, []models.WebexRecipient{result.src})...).
				Value(&result.dst),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Start Date").
				Description("YYYY-MM-DD format. Leave blank for immediate.").
				Value(&result.start).
				Validate(func(s string) error {
					t := strings.TrimSpace(s)
					if t == "" {
						return nil
					}

					if !isValidDate(t) {
						return errInvalidDateInput
					}

					return nil
				}),
			huh.NewInput().
				Title("End Date").
				Description("YYYY-MM-DD format. Leave blank for indefinite.").
				Value(&result.end).
				Validate(func(s string) error {
					t := strings.TrimSpace(s)
					if t == "" {
						return nil
					}

					if !isValidDate(t) {
						return errInvalidDateInput
					}

					if result.start != "" {
						st, _ := time.Parse("2006-01-02", result.start)
						et, _ := time.Parse("2006-01-02", t)
						if st.After(et) {
							return errEndEarlierThanStart
						}
					}

					return nil
				}),
			huh.NewConfirm().
				Title("Source User Keeps Copy").
				Negative("No").
				Affirmative("Yes").
				Value(&result.userKeeps),
		),
	).WithTheme(theme).WithShowHelp(false) // add +1 to height to account for not showing help
}

func fwdFormResToForm(res *fwdsFormResult) *models.NotifierForward {
	fwd := &models.NotifierForward{
		SourceID:      res.src.ID,
		DestID:        res.dst.ID,
		UserKeepsCopy: res.userKeeps,
		Enabled:       true,
	}

	if strings.TrimSpace(res.start) != "" {
		p, _ := time.Parse("2006-01-02", res.start)
		fwd.StartDate = &p
	}

	if strings.TrimSpace(res.end) != "" {
		p, _ := time.Parse("2006-01-02", res.end)
		fwd.EndDate = &p
	}

	return fwd
}
