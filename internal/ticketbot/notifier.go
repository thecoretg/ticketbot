package ticketbot

import (
	"fmt"
	"strings"
	"tctg-automation/internal/ticketbot/db"
)

const maxNoteLen = 300

//func (s *server) notifyNewTicket(noteID int) error {
//	note, err := s.dbHandler.GetTicketNote(noteID)
//	if err != nil {
//		return fmt.Errorf("getting ticket note: %w", err)
//	}
//
//	contactName, companyName := s.getNoteSenderNameAndCompany(note.ContactID, note.MemberID)
//
//}

func (s *server) getNoteSenderNameAndCompany(contactID, memberID *int) (string, string) {
	var contactName, companyName string
	var contact *db.Contact
	if contactID != nil {
		contact, _ = s.dbHandler.GetContact(*contactID)
		if contact == nil {
			return contactName, companyName
		}

		contactName = fullName(contact.FirstName, contact.LastName)
		if contact.CompanyID != nil {
			company, _ := s.dbHandler.GetCompany(*contact.CompanyID)
			if company != nil {
				companyName = company.Name
			}
		}

		return contactName, companyName

	} else if memberID != nil {
		companyName = "Internal"
		member, _ := s.dbHandler.GetMember(*memberID)
		if member != nil {
			contactName = fullName(member.FirstName, &member.LastName)
		}
		return contactName, companyName
	}

	return contactName, companyName
}

func fullName(first string, last *string) string {
	f := first
	if last != nil {
		f = first + " " + *last
	}

	return f
}

func formatNewTicketMsg(ticket *db.Ticket, contact *db.Contact, company *db.Company, note *db.TicketNote, noteSenderName string) string {
	msg := fmt.Sprintf("**New:** [%d](%s)", ticket.ID, ticketLink(ticket.ID))
	msg += fmt.Sprintf("\n**Company:** %s", company.Name)
	if contact != nil {
		msg += fmt.Sprintf(" / %s", fullName(contact.FirstName, contact.LastName))
	}

	if note != nil && note.Content != nil {
		msg += fmt.Sprintf("\n**Most Recent Note By:** %s", noteSenderName)
		msg += fmt.Sprintf("\n%s", quoteBlock(*note.Content, maxNoteLen))
	}

	return msg
}

func ticketLink(ticketID int) string {
	// TODO: Un-hardcode this
	return fmt.Sprintf("https://na.myconnectwise.net/v4_6_release/services/system_io/Service/fv_sr100_request.rails?service_recid=%d&companyName=securenetit", ticketID)
}

func quoteBlock(text string, maxLen int) string {
	if len(text) > maxLen {
		runes := []rune(text)
		if len(runes) > maxLen {
			text = string(runes[:maxLen]) + "..."
		}
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = ">" + line
	}

	return strings.Join(lines, "\n")
}
