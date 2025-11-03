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
	ticket      db.CwTicket
	company     db.CwCompany
	contact     db.CwContact
	owner       db.CwMember
	note        db.CwTicketNote
	board       db.CwBoard
	notifyRooms []db.WebexRoom
}

func (cl *Client) handleTickets(c *gin.Context) {
	w := &psa.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("unmarshaling connectwise webhook payload: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("received no ticket ID or zero"))
		return
	}

	switch w.Action {
	case "added", "updated":
		if err := cl.processTicket(c.Request.Context(), w.ID, w.Action, false); err != nil {
			c.Error(fmt.Errorf("ticket %d: adding or updating the ticket into data storage: %w", w.ID, err))
			return
		}

		c.Status(http.StatusOK)

	case "deleted":
		if err := cl.softDeleteTicket(c.Request.Context(), w.ID); err != nil {
			c.Error(fmt.Errorf("ticket %d: deleting ticket and its notes: %w", w.ID, err))
			return
		}

		c.Status(http.StatusOK)
	}
}

func (cl *Client) getTicketLock(ticketID int) *sync.Mutex {
	lockIface, _ := cl.ticketLocks.LoadOrStore(ticketID, &sync.Mutex{})
	return lockIface.(*sync.Mutex)
}

// processTicket serves as the primary handler for updating the data store with ticket data. It also will handle
// extra functionality such as ticket notifications.
func (cl *Client) processTicket(ctx context.Context, ticketID int, action string, bypassNotis bool) error {
	// Lock the ticket so that extra calls don't interfere. Due to the nature of Connectwise updates will often
	// result in other hooks and actions taking place, which means a ticket rarely only sends one webhook payload.
	lock := cl.getTicketLock(ticketID)
	if !lock.TryLock() {
		lock.Lock()
	}

	defer func() {
		lock.Unlock()
	}()

	// Get the current data for the ticket via the Connectwise API.
	// This will be used to compare for changes with the store ticket.
	cd, err := cl.getCwData(ticketID)
	if err != nil {
		return fmt.Errorf("getting data from connectwise: %w", err)
	}

	sd, err := cl.getStoredData(ctx, cd)
	if err != nil {
		return fmt.Errorf("getting or creating stored data: %w", err)
	}

	// Insert or update the ticket into the store if it didn't exist or if there were changes.
	p := cwDataToUpdateTicketParams(cd)
	sd.ticket, err = cl.Queries.UpsertTicket(ctx, p)
	if err != nil {
		return fmt.Errorf("updating ticket in store: %w", err)
	}

	// If a note exists and notifications are on, run the ticket notification action,
	// which checks if it meets message criteria and then notifies if valid.
	// AttemptNotify and the bypassNotis (used for preloads) acts as a hard block from even attempting.
	notified := false
	if cl.Config.AttemptNotify && sd.note.ID != 0 && !bypassNotis {
		notified, err = cl.runNotificationAction(ctx, action, cd, sd)
		if err != nil {
			return fmt.Errorf("running notifier: %w", err)
		}
	}

	// Log the ticket result regardless of what happened
	logTicketResult(action, notified, cl.testing.mockWebex, sd)
	return nil
}

func (cl *Client) runNotificationAction(ctx context.Context, action string, cd *cwData, sd *storedData) (bool, error) {
	notified := false
	if meetsMessageCriteria(action, sd) {
		// set notified first in case message fails - don't want to send duplicates regardless
		if err := cl.setNotified(ctx, sd.note.ID, true); err != nil {
			return false, fmt.Errorf("setting notified to true: %w", err)
		}

		if err := cl.makeAndSendMessages(ctx, action, cd, sd); err != nil {
			return false, fmt.Errorf("processing webex messages: %w", err)
		}
		notified = true
	}
	return notified, nil
}

func (cl *Client) softDeleteTicket(ctx context.Context, ticketID int) error {
	if err := cl.Queries.SoftDeleteTicket(ctx, ticketID); err != nil {
		return fmt.Errorf("soft deleting ticket: %w", err)
	}

	return nil
}

func (cl *Client) getCwData(ticketID int) (*cwData, error) {
	ticket, err := cl.CWClient.GetTicket(ticketID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting ticket: %w", err)
	}

	note, err := cl.CWClient.GetMostRecentTicketNote(ticketID)
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

func (cl *Client) getStoredData(ctx context.Context, cd *cwData) (*storedData, error) {
	// first check for or create board since it needs to exist before the ticket
	board, err := cl.ensureBoardInStore(ctx, cd.ticket.Board.ID)
	if err != nil {
		return nil, fmt.Errorf("ensuring board in store: %w", err)
	}

	company, err := cl.ensureCompanyInStore(ctx, cd.ticket.Company.ID)
	if err != nil {
		return nil, fmt.Errorf("ensuring company in store: %w", err)
	}

	contact := db.CwContact{}
	if cd.ticket.Contact.ID != 0 {
		contact, err = cl.ensureContactInStore(ctx, cd.ticket.Contact.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring contact in store: %w", err)
		}
	}

	owner := db.CwMember{}
	if cd.ticket.Owner.ID != 0 {
		owner, err = cl.ensureMemberInStore(ctx, cd.ticket.Owner.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring owner in store: %w", err)
		}
	}

	// check for, or create ticket
	ticket, err := cl.ensureTicketInStore(ctx, cd)
	if err != nil {
		return nil, fmt.Errorf("ensuring ticket in store: %w", err)
	}

	// start with empty note, use existing or created note if there is a note in the ticket
	note := db.CwTicketNote{}
	if cd.note.ID != 0 {
		note, err = cl.ensureNoteInStore(ctx, cd)
		if err != nil {
			return nil, fmt.Errorf("ensuring note in store: %w", err)
		}
	}

	cons, err := cl.Queries.ListNotifierConnectionsByBoard(ctx, board.ID)
	if err != nil {
		return nil, fmt.Errorf("getting rooms to notify: %w", err)
	}

	return &storedData{
		ticket:      ticket,
		company:     company,
		contact:     contact,
		owner:       owner,
		note:        note,
		board:       board,
		notifyRooms: roomsFromNotifiers(cons),
	}, nil
}

func (cl *Client) ensureTicketInStore(ctx context.Context, cd *cwData) (db.CwTicket, error) {
	ticket, err := cl.Queries.GetTicket(ctx, cd.ticket.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p := db.UpsertTicketParams{
				ID:        cd.ticket.ID,
				Summary:   cd.ticket.Summary,
				BoardID:   cd.ticket.Board.ID,
				OwnerID:   intToPtr(cd.ticket.Owner.ID),
				CompanyID: cd.ticket.Company.ID,
				ContactID: intToPtr(cd.ticket.Contact.ID),
				Resources: &cd.ticket.Resources,
				UpdatedBy: &cd.ticket.Info.UpdatedBy,
			}

			ticket, err = cl.Queries.UpsertTicket(ctx, p)
			if err != nil {
				return db.CwTicket{}, fmt.Errorf("inserting ticket into db: %w", err)
			}

			return ticket, nil
		} else {
			return db.CwTicket{}, fmt.Errorf("getting ticket from storage: %w", err)
		}
	}

	return ticket, nil
}

func logTicketResult(action string, notified, mockMsg bool, sd *storedData) {
	msg := "ticket processed"
	if mockMsg {
		msg = "ticket processed with webex mocking"
	}

	slog.Info(msg,
		"ticket_id", sd.ticket.ID,
		"action", action,
		"notified", notified)
}

// meetsMessageCriteria checks if a message would be allowed to send a notification,
// depending on if it was added or updated, if the note changed, and the board's notification settings.
func meetsMessageCriteria(action string, sd *storedData) bool {
	slog.Debug("checking message conditions", "action", action, "ticket_id", sd.ticket.ID, "note_id", sd.note.ID,
		"board_id", sd.board.ID, "already_notified", sd.note.Notified)
	meetsCrit := false
	if action == "added" {
		meetsCrit = roomsToNotifyExist(sd)
	}

	if action == "updated" {
		meetsCrit = !sd.note.Notified && roomsToNotifyExist(sd)
	}

	return meetsCrit
}

func roomsToNotifyExist(sd *storedData) bool {
	return sd.notifyRooms != nil && len(sd.notifyRooms) > 0
}

func cwDataToUpdateTicketParams(cd *cwData) db.UpsertTicketParams {
	return db.UpsertTicketParams{
		ID:        cd.ticket.ID,
		Summary:   cd.ticket.Summary,
		BoardID:   cd.ticket.Board.ID,
		OwnerID:   intToPtr(cd.ticket.Owner.ID),
		CompanyID: cd.ticket.Company.ID,
		ContactID: intToPtr(cd.ticket.Contact.ID),
		Resources: strToPtr(cd.ticket.Resources),
		UpdatedBy: strToPtr(cd.ticket.Info.UpdatedBy),
	}
}

func roomsFromNotifiers(notifiers []db.ListNotifierConnectionsByBoardRow) []db.WebexRoom {
	var rooms []db.WebexRoom
	for _, n := range notifiers {
		rooms = append(rooms, n.WebexRoom)
	}

	return rooms
}
