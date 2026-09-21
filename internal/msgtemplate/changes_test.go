package msgtemplate

import (
	"testing"

	"github.com/thecoretg/ticketbot/internal/ticketdiff"
	"github.com/thecoretg/ticketbot/models"
)

func TestFormatChanges(t *testing.T) {
	got := FormatChanges([]models.FieldChange{
		{Field: "status", Old: ticketdiff.Ref{ID: 1, Name: "New"}, New: ticketdiff.Ref{ID: 2, Name: "Assigned"}},
		{Field: "owner", Old: nil, New: ticketdiff.Ref{ID: 5, Name: "jdoe"}},
		{Field: "resources", Old: []string{"jdoe"}, New: []string{"jdoe", "asmith"}},
		{Field: "closedFlag", Old: false, New: true},
		{Field: "summary", Old: "", New: "Printer down"},
		{Field: "custom", Old: map[string]any{"id": float64(3)}, New: map[string]any{"name": "X"}},
	})
	want := "Status: New → Assigned · Owner: — → jdoe · Resources: jdoe → asmith, jdoe · Closed: no → yes · Summary: — → Printer down · custom: #3 → X"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if FormatChanges(nil) != "" {
		t.Error("no changes should render empty")
	}
}
