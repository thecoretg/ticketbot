package connectwise

import (
	"fmt"
)

const (
	ticketLinkBase = "https://na.myconnectwise.net/v4_6_release/services/system_io/Service/fv_sr100_request.rails?service_recid="
)

func ticketIdEndpoint(ticketId int) string {
	return fmt.Sprintf("service/tickets/%d", ticketId)
}

func ticketIdNotesEndpoint(ticketId int) string {
	return fmt.Sprintf("%s/notes", ticketIdEndpoint(ticketId))
}

func ticketIdNoteIdEndpoint(ticketId, noteId int) string {
	return fmt.Sprintf("%s/%d", ticketIdNotesEndpoint(ticketId), noteId)
}

func (c *Client) PostTicket(ticket *Ticket) (*Ticket, error) {
	return Post[Ticket](c, "service/tickets", ticket)
}

func (c *Client) ListTickets(params map[string]string) ([]Ticket, error) {
	return GetMany[Ticket](c, "service/tickets", params)
}

func (c *Client) GetTicket(ticketID int, params map[string]string) (*Ticket, error) {
	return GetOne[Ticket](c, ticketIdEndpoint(ticketID), params)
}

func (c *Client) PutTicket(ticketID int, ticket *Ticket) (*Ticket, error) {
	return Put[Ticket](c, ticketIdEndpoint(ticketID), ticket)
}

func (c *Client) PatchTicket(ticketID int, patchOps []PatchOp) (*Ticket, error) {
	return Patch[Ticket](c, ticketIdEndpoint(ticketID), patchOps)
}

func (c *Client) DeleteTicket(ticketID int) error {
	return Delete(c, ticketIdEndpoint(ticketID))
}

// ListServiceTicketNotesAll gets all ticket notes, regardless of if they have a time entry.
//
// This is most likely the one you want to use unless you consistently uncheck the time entry box.
func (c *Client) ListServiceTicketNotesAll(params map[string]string, ticketID int) ([]ServiceTicketNoteAll, error) {
	return GetMany[ServiceTicketNoteAll](c, ticketIdNotesEndpoint(ticketID), params)
}

func (c *Client) PostServiceTicketNote(ticketNote *ServiceTicketNote, ticketID int) (*ServiceTicketNote, error) {
	return Post[ServiceTicketNote](c, ticketIdNotesEndpoint(ticketID), ticketNote)
}

// ListServiceTicketNotes gets all notes that are not time entry.
//
// Not recommended since you will probably get what you need through ListServiceTicketNotes
func (c *Client) ListServiceTicketNotes(params map[string]string, ticketID int) ([]ServiceTicketNote, error) {
	return GetMany[ServiceTicketNote](c, ticketIdNotesEndpoint(ticketID), params)
}

func (c *Client) GetServiceTicketNote(noteID int, params map[string]string, ticketID int) (*ServiceTicketNote, error) {
	return GetOne[ServiceTicketNote](c, ticketIdNoteIdEndpoint(ticketID, noteID), params)
}

func (c *Client) PutServiceTicketNote(noteID int, ticketNote *ServiceTicketNote, ticketID int) (*ServiceTicketNote, error) {
	return Put[ServiceTicketNote](c, ticketIdNoteIdEndpoint(ticketID, noteID), ticketNote)
}

func (c *Client) PatchServiceTicketNote(noteID int, patchOps []PatchOp, ticketID int) (*ServiceTicketNote, error) {
	return Patch[ServiceTicketNote](c, ticketIdNoteIdEndpoint(ticketID, noteID), patchOps)
}

func (c *Client) DeleteServiceTicketNote(noteID int, ticketID int) error {
	return Delete(c, ticketIdNoteIdEndpoint(ticketID, noteID))
}

func (c *Client) GetMostRecentTicketNote(ticketID int) (*ServiceTicketNote, error) {
	p := map[string]string{
		"orderBy":  "id desc",
		"pageSize": "1",
	}

	notes, err := c.ListServiceTicketNotesAll(p, ticketID)
	if err != nil {
		return nil, fmt.Errorf("listing service notes: %w", err)
	}

	if len(notes) != 1 {
		return nil, nil
	}

	note, err := c.GetServiceTicketNote(notes[0].ID, nil, ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting details for note: %w", err)
	}

	return note, nil
}

func MarkdownInternalTicketLink(ticketID int, companyID string) string {
	return fmt.Sprintf("[%d](%s)", ticketID, InternalTicketLink(ticketID, companyID))
}

func InternalTicketLink(ticketID int, companyID string) string {
	return fmt.Sprintf("%s%d&companyName=%s", ticketLinkBase, ticketID, companyID)
}
