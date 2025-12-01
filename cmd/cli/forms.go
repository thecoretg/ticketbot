package main

import (
	"github.com/charmbracelet/huh"
)

var yesNoOpts = []huh.Option[bool]{
	{
		Key:   "Yes",
		Value: true,
	},
	{
		Key:   "No",
		Value: false,
	},
}

// form is a wrapper to huh.NewForm which automatically assigns defaults.
func form(groups ...*huh.Group) *huh.Form {
	f := huh.NewForm(groups...).
		WithTheme(huh.ThemeBase())

	return f
}
