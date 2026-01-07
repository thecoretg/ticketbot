package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

type (
	fwdsModel struct {
		SDKClient  *sdk.Client
		initRan    bool
		fwdsLoaded bool
		table      table.Model
		form       *huh.Form
		spinner    spinner.Model
		formResult *fwdsFormResult
		status     fwdsModelStatus

		availWidth  int
		availHeight int
		fwds        []models.NotifierForwardFull
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

	refreshFwdsMsg    struct{}
	gotFwdsMsg        struct{ fwds []models.NotifierForwardFull }
	updateFmStatusMsg struct{ status fwdsModelStatus }
	fwdsModelStatus   int
)

var (
	errInvalidDateInput    = errors.New("valid date in format YYYY-MM-DD required")
	errEndEarlierThanStart = errors.New("end time cannot be before start time")
)

const (
	fwdStatusInitializing fwdsModelStatus = iota
	fwdStatusTable
	fwdStatusLoadingData
	fwdStatusEntry
	fwdStatusRefreshing
)

func newFwdsModel(cl *sdk.Client) *fwdsModel {
	s := spinner.New()
	s.Spinner = spinner.Line

	return &fwdsModel{
		SDKClient:  cl,
		fwds:       []models.NotifierForwardFull{},
		table:      newTable(),
		formResult: &fwdsFormResult{},
		spinner:    s,
	}
}

func (fm *fwdsModel) Init() tea.Cmd {
	fm.initRan = true
	return tea.Batch(fm.spinner.Tick, fm.getFwds())
}

func (fm *fwdsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, allKeys.newItem) && fm.status == fwdStatusTable:
			return fm, tea.Batch(updateFwdsStatus(fwdStatusLoadingData), fm.prepareForm())
		case key.Matches(msg, allKeys.deleteItem) && fm.status == fwdStatusTable:
			if len(fm.fwds) > 0 {
				fwd := fm.fwds[fm.table.Cursor()]
				return fm, tea.Batch(updateFwdsStatus(fwdStatusRefreshing), fm.deleteFwd(fwd.ID))
			}
		}
	case resizeModelsMsg:
		fm.availWidth = msg.w
		fm.availHeight = msg.h
		fm.setModuleDimensions()

	case refreshFwdsMsg:
		return fm, fm.getFwds()

	case gotFwdsMsg:
		fm.fwds = msg.fwds
		fm.fwdsLoaded = true
		return fm, tea.Batch(updateFwdsStatus(fwdStatusTable), fm.setRows())

	case fwdsFormDataMsg:
		fm.formResult = &fwdsFormResult{}
		fm.form = fwdEntryForm(msg.recips, fm.formResult)
		return fm, tea.Batch(updateFwdsStatus(fwdStatusEntry), fm.form.Init())

	case updateFmStatusMsg:
		fm.status = msg.status
	}

	var cmds []tea.Cmd
	switch fm.status {
	case fwdStatusEntry:
		fm.setFormHeight(fm.availHeight)
		form, cmd := fm.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			fm.form = f
		}

		cmds = append(cmds, cmd)
		switch fm.form.State {
		case huh.StateAborted:
			cmds = append(cmds, updateFwdsStatus(fwdStatusTable))

		case huh.StateCompleted:
			res := fm.formResult
			fwd := fwdFormResToForm(res)
			cmds = append(cmds, fm.submitFwd(fwd), updateFwdsStatus(fwdStatusRefreshing))
		}

	default:
		var cmd tea.Cmd
		fm.table, cmd = fm.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	fm.spinner, cmd = fm.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return fm, tea.Batch(cmds...)
}

func (fm *fwdsModel) View() string {
	switch fm.status {
	case fwdStatusInitializing:
		return fillSpaceCentered(useSpinner(fm.spinner, "Loading forwards..."), fm.availWidth, fm.availHeight)
	case fwdStatusRefreshing:
		return fillSpaceCentered(useSpinner(fm.spinner, "Refreshing..."), fm.availWidth, fm.availHeight)
	case fwdStatusTable:
		return fm.table.View()
	case fwdStatusLoadingData:
		return fillSpaceCentered(useSpinner(fm.spinner, "Loading form data..."), fm.availWidth, fm.availHeight)
	case fwdStatusEntry:
		return fm.form.View()
	}

	return fm.table.View()
}

func (fm *fwdsModel) setModuleDimensions() {
	fm.setTableDimensions(fm.availWidth, fm.availHeight)
	if fm.form != nil {
		fm.setFormHeight(fm.availHeight)
	}
}

func (fm *fwdsModel) setTableDimensions(w, h int) {
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
		recips, err := fm.SDKClient.ListRecipients()
		if err != nil {
			return errMsg{fmt.Errorf("listing recipients: %w", err)}
		}

		if len(recips) == 0 {
			currentErr = fmt.Errorf("no recipients available to create forward")
			return updateFmStatusMsg{status: fwdStatusTable}
		}

		sortRecips(recips)

		return fwdsFormDataMsg{
			recips: recips,
		}
	}
}

func (fm *fwdsModel) submitFwd(fwd *models.NotifierForward) tea.Cmd {
	return func() tea.Msg {
		_, err := fm.SDKClient.CreateUserForward(fwd)
		if err != nil {
			currentErr = fmt.Errorf("creating notifier forward: %w", err)
		}

		return refreshFwdsMsg{}
	}
}

func (fm *fwdsModel) deleteFwd(id int) tea.Cmd {
	return func() tea.Msg {
		if err := fm.SDKClient.DeleteUserForward(id); err != nil {
			currentErr = fmt.Errorf("deleting notifier forward: %w", err)
		}

		return refreshFwdsMsg{}
	}
}

func (fm *fwdsModel) getFwds() tea.Cmd {
	return func() tea.Msg {
		fwds, err := fm.SDKClient.ListUserForwards()
		if err != nil {
			currentErr = fmt.Errorf("listing notifier forwards: %w", err)
		}

		return gotFwdsMsg{fwds: fwds}
	}
}

func (fm *fwdsModel) setRows() tea.Cmd {
	// TODO: sort fwds
	fm.table.SetRows(fwdsToRows(fm.fwds))
	fm.table.SetCursor(0)
	return nil
}

func fwdsToRows(fwds []models.NotifierForwardFull) []table.Row {
	if len(fwds) == 0 {
		return []table.Row{
			{
				"NO", "FWDS", "FOUND", ":(", "",
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

func updateFwdsStatus(status fwdsModelStatus) tea.Cmd {
	return func() tea.Msg {
		return updateFmStatusMsg{status: status}
	}
}
