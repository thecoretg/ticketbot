package ticketbot

import (
	"fmt"

	"github.com/thecoretg/ticketbot/internal/service/journal"
	"github.com/thecoretg/ticketbot/models"
)

// buildChangeEvents compares the ticket's previous journal snapshot to the newly
// synced FullTicket and returns info-level events for each field that changed, plus
// a note event if the latest note is newer than the last run.
func buildChangeEvents(old *models.TicketJournal, full *models.FullTicket) []models.JournalEvent {
	if full == nil {
		return nil
	}

	var events []models.JournalEvent

	newStatus := full.Status.Name
	newContact := journal.ContactName(full.Contact)
	newOwner := journal.MemberName(full.Owner)
	newType := journal.TypeName(full.Type)
	newSubtype := journal.SubTypeName(full.SubType)
	newItem := journal.ItemName(full.Item)
	newResources := journal.ResourceNames(full.Resources)

	if old != nil {
		if old.StatusName != newStatus {
			events = append(events, changeEvent("Status", old.StatusName, newStatus))
		}
		if old.ContactName != newContact {
			events = append(events, changeEvent("Contact", old.ContactName, newContact))
		}
		if old.OwnerName != newOwner {
			events = append(events, changeEvent("Owner", old.OwnerName, newOwner))
		}
		if old.TypeName != newType {
			events = append(events, changeEvent("Type", old.TypeName, newType))
		}
		if old.SubtypeName != newSubtype {
			events = append(events, changeEvent("Subtype", old.SubtypeName, newSubtype))
		}
		if old.ItemName != newItem {
			events = append(events, changeEvent("Item", old.ItemName, newItem))
		}
		if old.ResourceNames != newResources {
			events = append(events, changeEvent("Resources", old.ResourceNames, newResources))
		}
	}

	if full.LatestNote != nil {
		noteIsNew := old == nil || full.LatestNote.AddedOn.After(old.LastRun)
		if noteIsNew {
			events = append(events, noteEvent(full.LatestNote))
		}
	}

	return events
}

func changeEvent(field, from, to string) models.JournalEvent {
	if from == "" {
		from = "(none)"
	}
	if to == "" {
		to = "(none)"
	}
	return models.JournalEvent{
		Text:   fmt.Sprintf("%s: %s → %s", field, from, to),
		Status: models.JournalOK,
	}
}

func noteEvent(note *models.FullTicketNote) models.JournalEvent {
	sender := "(unknown)"
	senderType := ""
	if note.Member != nil {
		sender = journal.MemberName(note.Member)
		senderType = "member"
	} else if note.Contact != nil {
		sender = journal.ContactName(note.Contact)
		senderType = "contact"
	}

	label := sender
	if senderType != "" {
		label = fmt.Sprintf("%s (%s)", sender, senderType)
	}

	var tags []string
	if note.Flags != nil {
		if note.Flags.Discussion {
			tags = append(tags, "Discussion")
		}
		if note.Flags.Internal {
			tags = append(tags, "Internal")
		}
		if note.Flags.Resolution {
			tags = append(tags, "Resolution")
		}
	}

	return models.JournalEvent{
		Text:   "Note by " + label,
		Status: models.JournalOK,
		Tags:   tags,
	}
}
