package transformer

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

// renderTemplate parses and executes a single Go template string against data.
// text/template (not html/template) is used because these are plain Connectwise
// field values, not HTML. A missing struct field surfaces as an execute error,
// so typos are caught at rule-save validation time.
func renderTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("field").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// renderParams walks the string fields of a params struct that carry a `tmpl`
// tag and replaces each value with its rendered output against the ticket.
func renderParams(p Params, t *psa.Ticket) error {
	v := reflect.ValueOf(p)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("params must be a non-nil pointer, got %T", p)
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("params must point to a struct, got %s", v.Kind())
	}

	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if _, ok := field.Tag.Lookup("tmpl"); !ok {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() != reflect.String || !fv.CanSet() {
			continue
		}

		rendered, err := renderTemplate(fv.String(), t)
		if err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
		fv.SetString(rendered)
	}

	return nil
}

// validateTemplates parses (but does not execute) every templated field of a
// params struct, returning the first syntax error. Used at rule-save time.
func validateTemplates(p Params) error {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("params must be a struct, got %s", v.Kind())
	}

	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if _, ok := field.Tag.Lookup("tmpl"); !ok {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() != reflect.String {
			continue
		}
		if _, err := template.New("field").Option("missingkey=error").Parse(fv.String()); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}

	return nil
}
