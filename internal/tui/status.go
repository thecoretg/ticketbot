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
)

func (s subModelStatus) inForm() bool {
	return s == statusEntry || s == statusConfirm
}

func (s subModelStatus) quittable() bool {
	switch s {
	case statusInit, statusMain, statusLoadingFormData:
		return true
	default:
		return false
	}
}
