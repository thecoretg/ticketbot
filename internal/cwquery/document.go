package cwquery

import (
	"encoding/json"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// Changes describes what an intake pass observed, for the pseudo-fields NewDocument adds.
type Changes struct {
	Fields  []models.FieldChange
	NewNote bool
	IsNew   bool
}

// NewDocument builds the evaluation document for a ticket: the ticket's JSON form plus pseudo-fields:
//
//   - latestNote: the most recent ticket note (nil when there is none), so conditions such as
//     `latestNote/text contains 'x'` or `latestNote/internalAnalysisFlag = true` work.
//   - newNote: true when the latest note first appeared in this intake, so `newNote = true`
//     separates "someone replied" from "a field changed".
//   - isNew: true when ticketbot is seeing the ticket for the first time, so one branch can tell
//     a brand-new ticket from an update.
//   - changed: a map of field name → true for each changed field, so `changed/status = true`
//     matches only when the status changed in this intake.
//   - old: the previous value of each changed field, so `old/status/name = 'New'` matches a
//     ticket that just left New. Reference fields are {id, name}; resources is a comma-separated
//     string like the ticket's own resources field.
func NewDocument(t *psa.Ticket, note *psa.ServiceTicketNote, ch Changes) map[string]any {
	doc := toMap(t)
	if doc == nil {
		doc = map[string]any{}
	}

	if note != nil {
		doc["latestNote"] = toMap(note)
	} else {
		doc["latestNote"] = nil
	}
	doc["newNote"] = ch.NewNote
	doc["isNew"] = ch.IsNew

	changed := make(map[string]any, len(ch.Fields))
	old := make(map[string]any, len(ch.Fields))
	for _, c := range ch.Fields {
		changed[c.Field] = true
		old[c.Field] = toValue(c.Old)
	}
	doc["changed"] = changed
	doc["old"] = old

	return doc
}

func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}

	return m
}

// toValue normalizes a FieldChange value to the JSON-decoded shapes Eval understands.
func toValue(v any) any {
	if s, ok := v.([]string); ok {
		return strings.Join(s, ", ")
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}

	return out
}
