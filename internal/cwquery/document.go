package cwquery

import (
	"encoding/json"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

// NewDocument builds the evaluation document for a ticket: the ticket's JSON form plus two
// pseudo-fields:
//
//   - latestNote: the most recent ticket note (nil when there is none), so conditions such as
//     `latestNote/text contains 'x'` or `latestNote/internalAnalysisFlag = true` work.
//   - changed: a map of field name → true for each field in changed, so `changed/status = true`
//     matches only when the status changed in this intake.
func NewDocument(t *psa.Ticket, note *psa.ServiceTicketNote, changed []string) map[string]any {
	doc := toMap(t)
	if doc == nil {
		doc = map[string]any{}
	}

	if note != nil {
		doc["latestNote"] = toMap(note)
	} else {
		doc["latestNote"] = nil
	}

	ch := make(map[string]any, len(changed))
	for _, f := range changed {
		ch[f] = true
	}
	doc["changed"] = ch

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
