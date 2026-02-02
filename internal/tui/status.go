package tui

type (
	subModelStatus int
)

const (
	statusInit subModelStatus = iota
	statusMain
	statusLoadingFormData
	statusEntry
	statusConfirm
	statusRefresh
	statusShowKey
	statusSyncing
	statusError
)

func (s subModelStatus) inForm() bool {
	return s == statusEntry || s == statusConfirm
}
