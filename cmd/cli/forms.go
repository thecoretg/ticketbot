package main

import (
	"github.com/charmbracelet/huh"
)

// form is a wrapper to huh.NewForm which automatically assigns defaults.
func form(groups ...*huh.Group) *huh.Form {
	f := huh.NewForm(groups...).
		WithTheme(huh.ThemeBase())

	return f
}
