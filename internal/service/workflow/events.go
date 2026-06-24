package workflow

import (
	"fmt"
	"strings"

	"github.com/thecoretg/ticketbot/models"
)

// actionLabels maps action types to friendly names for the ticket journal.
var actionLabels = map[string]string{
	models.WorkflowActionTicketUpdate:      "Update Ticket",
	models.WorkflowActionAddNote:           "Add Note",
	models.WorkflowActionSendMessage:       "Send Notification",
	models.WorkflowActionSkipNotifications: "Skip Default Notifications",
	models.WorkflowActionAddResource:       "Add Resource",
	models.WorkflowActionAddEmailCc:        "Add Email CC",
}

func actionLabel(t string) string {
	if l, ok := actionLabels[t]; ok {
		return l
	}
	return t
}

func errEvent(format string, args ...any) models.JournalEvent {
	return models.JournalEvent{Text: fmt.Sprintf(format, args...), Status: models.JournalError}
}

// matchedEvent describes a workflow that matched, including a friendly summary of
// the conditions that were met.
func matchedEvent(wf *models.Workflow) models.JournalEvent {
	text := fmt.Sprintf("Matched workflow %q", wf.Name)
	if cond := summarizeConditions(wf.Root); cond != "" {
		text += " — " + cond
	}
	return models.JournalEvent{Text: text, Status: models.JournalInfo}
}

// actionEvent turns an action's Change into a friendly timeline line.
func actionEvent(actionType string, c Change) models.JournalEvent {
	status := models.JournalOK
	if !c.Applied {
		status = models.JournalInfo
	}
	if c.Simulated && c.Applied {
		status = models.JournalSkip
	}

	// did picks the verb mood for the applied branch: past tense for a real run,
	// "Would …" for a simulation.
	did := func(real, would string) string {
		if c.Simulated {
			return would
		}
		return real
	}

	var text string
	switch actionType {
	case models.WorkflowActionTicketUpdate:
		if c.Applied {
			text = did("Updated ticket", "Would update ticket")
			if c.Field != "" && c.Field != "ticket" {
				text += " (" + c.Field + ")"
			}
		} else {
			text = "Ticket already up to date"
		}
	case models.WorkflowActionAddNote:
		if c.Applied {
			text = did("Added note", "Would add note")
		} else {
			text = "No note added"
		}
	case models.WorkflowActionSendMessage:
		if c.Applied {
			text = did("Sent notification to ", "Would send notification to ") + c.To
		} else {
			text = "No notification sent"
		}
	case models.WorkflowActionSkipNotifications:
		text = did("Skipped default notifications", "Would skip default notifications")
		status = models.JournalSkip
	case models.WorkflowActionAddResource:
		if c.Applied {
			text = did("Added resource ", "Would add resource ") + c.To
		} else {
			text = "Resource already present"
		}
	case models.WorkflowActionAddEmailCc:
		if c.Applied {
			text = did("Added email CC ", "Would add email CC ") + c.To
		} else {
			text = "Email already on CC list"
		}
	default:
		text = actionLabel(actionType)
	}

	return models.JournalEvent{Text: text, Status: status, Simulated: c.Simulated}
}

// summarizeConditions renders a condition tree as a compact human-readable string,
// e.g. (Summary contains "printer" AND Status equals "open") OR Company is any of
// "Acme,Globex". Empty/nil groups yield "".
func summarizeConditions(g *models.ConditionGroup) string {
	if g == nil || len(g.Children) == 0 {
		return ""
	}

	joiner := " AND "
	if g.Operator == models.GroupOpOr {
		joiner = " OR "
	}

	parts := make([]string, 0, len(g.Children))
	for _, n := range g.Children {
		switch {
		case n.Group != nil:
			if sub := summarizeConditions(n.Group); sub != "" {
				parts = append(parts, "("+sub+")")
			}
		case n.Condition != nil:
			c := n.Condition
			// Boolean conditions (is_true/is_false) carry no value to quote.
			if c.Operator == "is_true" || c.Operator == "is_false" {
				parts = append(parts, fmt.Sprintf("%s %s", conditionFieldLabel(c.Field), operatorLabel(c.Operator)))
			} else {
				parts = append(parts, fmt.Sprintf("%s %s %q", conditionFieldLabel(c.Field), operatorLabel(c.Operator), c.Value))
			}
		}
	}
	return strings.Join(parts, joiner)
}

// conditionFieldLabels / operatorLabels give friendly names for the journal
// summary; they mirror the labels shown in the admin builder.
var conditionFieldLabels = map[string]string{
	"summary":               "Summary",
	"company_name":          "Company Name",
	"company_identifier":    "Company ID",
	"contact_name":          "Contact Name",
	"status_name":           "Status",
	"board_name":            "Board Name",
	"type_name":             "Type",
	"subtype_name":          "Subtype",
	"priority_name":         "Priority",
	"source_name":           "Source",
	"last_note_text":        "Last Note Text",
	"last_note_sender":      "Last Note Sender",
	"last_note_type":        "Last Note Type",
	"customer_updated_flag": "Customer Updated",
}

var operatorLabels = map[string]string{
	"contains":     "contains",
	"not_contains": "does not contain",
	"equals":       "equals",
	"not_equals":   "does not equal",
	"starts_with":  "starts with",
	"ends_with":    "ends with",
	"is_any_of":    "is any of",
	"is_none_of":   "is none of",
	"is_true":      "is on",
	"is_false":     "is off",
}

func conditionFieldLabel(f string) string {
	if l, ok := conditionFieldLabels[f]; ok {
		return l
	}
	return f
}

func operatorLabel(o string) string {
	if l, ok := operatorLabels[o]; ok {
		return l
	}
	return o
}
