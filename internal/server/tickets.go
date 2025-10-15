package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/psa"
)

type cwData struct {
	ticket *psa.Ticket
	note   *psa.ServiceTicketNote
}

type storedData struct {
	ticket  db.CwTicket
	company db.CwCompany
	contact db.CwContact
	owner   db.CwMember
	note    db.CwTicketNote
	board   db.CwBoard
}

func (s *Server) addHooksGroup() {
	hooks := s.GinEngine.Group("/hooks")
	cw := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(s.Config.General.ExitOnError))
	cw.POST("/tickets", s.handleTickets)
}

func (s *Server) handleTickets(c *gin.Context) {
	w := &psa.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("unmarshaling connectwise webhook payload: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("received no ticket ID or zero"))
		return
	}

	slog.Info("received payload from connectwise", "ticket_id", w.ID, "action", w.Action)
	switch w.Action {
	case "added", "updated":
		if err := s.processTicket(c.Request.Context(), w.ID, w.Action, s.Config.Messages.AttemptNotify); err != nil {
			c.Error(fmt.Errorf("ticket %d: adding or updating the ticket into data storage: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)

	case "deleted":
		if err := s.softDeleteTicket(c.Request.Context(), w.ID); err != nil {
			c.Error(fmt.Errorf("ticket %d: deleting ticket and its notes: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *Server) getTicketLock(ticketID int) *sync.Mutex {
	lockIface, _ := s.ticketLocks.LoadOrStore(ticketID, &sync.Mutex{})
	return lockIface.(*sync.Mutex)
}

// processTicket serves as the primary handler for updating the data store with ticket data. It also will handle
// extra functionality such as ticket notifications.
func (s *Server) processTicket(ctx context.Context, ticketID int, action string, attemptNotify bool) error {
	// Lock the ticket so that extra calls don't interfere. Due to the nature of Connectwise updates will often
	// result in other hooks and actions taking place, which means a ticket rarely only sends one webhook payload.
	lock := s.getTicketLock(ticketID)
	if !lock.TryLock() {
		lock.Lock()
	}

	defer func() {
		lock.Unlock()
	}()

	// Get the current data for the ticket via the Connectwise API.
	// This will be used to compare for changes with the store ticket.
	cwData, err := s.getCwData(ticketID)
	if err != nil {
		return fmt.Errorf("getting data from connectwise: %w", err)
	}

	storedData, err := s.getStoredData(ctx, cwData, attemptNotify)
	if err != nil {
		return fmt.Errorf("getting or creating stored data: %w", err)
	}

	// Insert or update the ticket into the store if it didn't exist or if there were changes.
	p := cwDataToUpdateTicketParams(cwData)
	storedData.ticket, err = s.Queries.UpdateTicket(ctx, p)
	if err != nil {
		return fmt.Errorf("updating ticket in store: %w", err)
	}

	// If a note exists, run the ticket notification action, which checks if it meets message
	// criteria and then notifies if valid
	if storedData.note.ID != 0 {
		if err := s.runNotificationAction(ctx, action, cwData, storedData, attemptNotify); err != nil {
			return fmt.Errorf("running notification action: %w", err)
		}
	}

	// Log the ticket result regardless of what happened
	s.logTicketResult(action, storedData)
	return nil
}

func (s *Server) runNotificationAction(ctx context.Context, action string, cwData *cwData, storedData *storedData, attemptNotify bool) error {
	if attemptNotify {
		if meetsMessageCriteria(action, storedData) {
			// set notified first in case message fails - don't want to send duplicates regardless
			if err := s.setNotified(ctx, storedData.note.ID, true); err != nil {
				return fmt.Errorf("setting notified to true: %w", err)
			}

			if err := s.makeAndSendWebexMsgs(ctx, action, cwData, storedData); err != nil {
				return fmt.Errorf("processing webex messages: %w", err)
			}
		}
	} else {
		if err := s.setSkippedNotify(ctx, storedData.note.ID, true); err != nil {
			return fmt.Errorf("setting notify skipped: %w", err)
		}
	}

	return nil
}

func (s *Server) softDeleteTicket(ctx context.Context, ticketID int) error {
	if err := s.Queries.SoftDeleteTicket(ctx, ticketID); err != nil {
		return fmt.Errorf("soft deleting ticket: %w", err)
	}
	slog.Debug("ticket soft deleted", "ticket_id", ticketID)

	return nil
}

func (s *Server) getCwData(ticketID int) (*cwData, error) {
	ticket, err := s.CWClient.GetTicket(ticketID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting ticket: %w", err)
	}

	note, err := s.CWClient.GetMostRecentTicketNote(ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting most recent note: %w", err)
	}

	if note == nil {
		note = &psa.ServiceTicketNote{}
	}

	return &cwData{
		ticket: ticket,
		note:   note,
	}, nil
}

func (s *Server) getStoredData(ctx context.Context, cwData *cwData, skipNotify bool) (*storedData, error) {
	// first check for or create board since it needs to exist before the ticket
	board, err := s.ensureBoardInStore(ctx, cwData)
	if err != nil {
		return nil, fmt.Errorf("ensuring board in store: %w", err)
	}

	company, err := s.ensureCompanyInStore(ctx, cwData.ticket.Company.ID)
	if err != nil {
		return nil, fmt.Errorf("ensuring company in store: %w", err)
	}

	contact := db.CwContact{}
	if cwData.ticket.Contact.ID != 0 {
		contact, err = s.ensureContactInStore(ctx, cwData.ticket.Contact.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring contact in store: %w", err)
		}
	}

	owner := db.CwMember{}
	if cwData.ticket.Owner.ID != 0 {
		owner, err = s.ensureMemberInStore(ctx, cwData.ticket.Owner.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring owner in store: %w", err)
		}
	}

	// check for, or create ticket
	ticket, err := s.ensureTicketInStore(ctx, cwData)
	if err != nil {
		return nil, fmt.Errorf("ensuring ticket in store: %w", err)
	}

	// start with empty note, use existing or created note if there is a note in the ticket
	note := db.CwTicketNote{}
	if cwData.note.ID != 0 {
		note, err = s.ensureNoteInStore(ctx, cwData, skipNotify)
		if err != nil {
			return nil, fmt.Errorf("ensuring note in store: %w", err)
		}
	}

	return &storedData{
		ticket:  ticket,
		company: company,
		contact: contact,
		owner:   owner,
		note:    note,
		board:   board,
	}, nil
}

func (s *Server) ensureTicketInStore(ctx context.Context, cwData *cwData) (db.CwTicket, error) {
	ticket, err := s.Queries.GetTicket(ctx, cwData.ticket.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p := db.InsertTicketParams{
				ID:        cwData.ticket.ID,
				Summary:   cwData.ticket.Summary,
				BoardID:   cwData.ticket.Board.ID,
				OwnerID:   intToPtr(cwData.ticket.Owner.ID),
				CompanyID: cwData.ticket.Company.ID,
				ContactID: intToPtr(cwData.ticket.Contact.ID),
				Resources: &cwData.ticket.Resources,
				UpdatedBy: &cwData.ticket.Info.UpdatedBy,
			}

			slog.Debug("created insert ticket params", "id", p.ID, "summary", p.Summary, "board_id", p.BoardID, "owner_id", p.OwnerID, "company_id", p.CompanyID, "contact_id", p.ContactID, "resources", p.Resources, "updated_by", p.UpdatedBy)

			ticket, err = s.Queries.InsertTicket(ctx, p)
			if err != nil {
				return db.CwTicket{}, fmt.Errorf("inserting ticket into db: %w", err)
			}

			slog.Debug("inserted ticket into db", "ticket_id", ticket.ID, "summary", ticket.Summary)
			return ticket, nil
		} else {
			return db.CwTicket{}, fmt.Errorf("getting ticket from storage: %w", err)
		}
	}

	slog.Debug("got existing ticket from store", "ticket_id", ticket.ID, "summary", ticket.Summary)
	return ticket, nil
}

func (s *Server) logTicketResult(action string, storedData *storedData) {
	slog.Info("ticket processed",
		"ticket_id", storedData.ticket.ID,
		"action", action,
		"attempt_notify", s.Config.Messages.AttemptNotify,
		"notified", storedData.note.Notified,
		"skipped_notify", storedData.note.SkippedNotify)
}

// meetsMessageCriteria checks if a message would be allowed to send a notification,
// depending on if it was added or updated, if the note changed, and the board's notification settings.
func meetsMessageCriteria(action string, storedData *storedData) bool {
	if action == "added" {
		return storedData.board.NotifyEnabled
	}

	if action == "updated" {
		return !storedData.note.Notified && storedData.board.NotifyEnabled
	}

	return false
}

func cwDataToUpdateTicketParams(cwData *cwData) db.UpdateTicketParams {
	return db.UpdateTicketParams{
		ID:        cwData.ticket.ID,
		Summary:   cwData.ticket.Summary,
		BoardID:   cwData.ticket.Board.ID,
		OwnerID:   intToPtr(cwData.ticket.Owner.ID),
		CompanyID: cwData.ticket.Company.ID,
		ContactID: intToPtr(cwData.ticket.Contact.ID),
		Resources: strToPtr(cwData.ticket.Resources),
		UpdatedBy: strToPtr(cwData.ticket.Info.UpdatedBy),
	}
}
