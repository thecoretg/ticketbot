package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"tctg-automation/internal/ticketbot/db"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/util"
)

func (s *server) processTicketPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("unmarshaling ConnectWise webhook payload: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("ticket ID cannot be 0"))
		return
	}

	slog.Info("received ticket webhook", "id", w.ID, "action", w.Action)
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteTicket(w.ID); err != nil {
			c.Error(fmt.Errorf("deleting ticket %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processTicketUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Info("ticket was deleted externally", "id", w.ID)
				if err := s.dbHandler.DeleteTicket(w.ID); err != nil {
					c.Error(fmt.Errorf("deleting ticket %d after external deletion: %w", w.ID, err))
					return
				}
				c.Status(http.StatusNoContent)
				return
			}

			c.Error(fmt.Errorf("processing ticket %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	}
}

func (s *server) processTicketUpdate(ctx context.Context, ticketID int) error {
	cwt, err := s.cwClient.GetTicket(ctx, ticketID, nil)
	if err != nil {
		return checkCWError("getting ticket info via CW API", "ticket", err, ticketID)
	}

	if err := s.ensureBoardExists(cwt.Board.ID, cwt.Board.Name); err != nil {
		return fmt.Errorf("ensuring board exists: %w", err)
	}

	if err := s.ensureStatusExists(ctx, cwt.Status.ID, cwt.Board.ID, cwt.Board.Name); err != nil {
		return fmt.Errorf("ensuring status exists: %w", err)
	}

	if err := s.ensureCompanyExists(cwt.Company.ID, cwt.Company.Name); err != nil {
		return fmt.Errorf("ensuring company exists: %w", err)
	}

	if cwt.Contact.ID != 0 {
		if err := s.ensureContactExists(ctx, cwt.Contact.ID); err != nil {
			return fmt.Errorf("ensuring contact exists: %w", err)
		}
	}

	// see if ticket already exists in db
	ticket, err := s.dbHandler.GetTicket(ticketID)
	if err != nil {
		return fmt.Errorf("getting existing ticket from DB: %w", err)
	}

	noteID, err := s.getMostRecentNote(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("getting most recent note: %w", err)
	}

	if ticket == nil {
		ticket = db.NewTicket(ticketID, cwt.Board.ID, cwt.Status.ID, cwt.Company.ID, cwt.Contact.ID, 0, cwt.Owner.ID, cwt.Summary, cwt.Resources, cwt.Info.DateEntered, cwt.Info.LastUpdated, cwt.ClosedDate, cwt.ClosedFlag)
		if err := s.dbHandler.UpsertTicket(ticket); err != nil {
			return fmt.Errorf("creating new ticket in db: %w", err)
		}
	}

	if noteID != 0 {
		if err := s.ensureTicketNoteExists(ctx, ticketID, noteID); err != nil {
			return fmt.Errorf("ensuring ticket note exists: %w", err)
		}

		// re fetch the ticket just in case of race conditions
		ticket, err = s.dbHandler.GetTicket(ticketID)
		if err != nil {
			return fmt.Errorf("getting most recent update of ticket: %w", err)
		}

		if ticket.LatestNote == nil || *ticket.LatestNote != noteID {
			ticket.LatestNote = util.IntToPtr(noteID)
			if err := s.dbHandler.UpsertTicket(ticket); err != nil {
				return fmt.Errorf("processing ticket in db: %w", err)
			}
		}
	}

	return nil
}

func (s *server) getMostRecentNote(ctx context.Context, ticketID int) (int, error) {
	p := &connectwise.QueryParams{OrderBy: "_info/dateEntered desc", PageSize: 1000}
	n, err := s.cwClient.ListServiceTicketNotes(ctx, ticketID, p)
	if err != nil {
		return 0, checkCWError("listing ticket notes", "ticket", err, ticketID)
	}

	for _, note := range n {
		if note.Text != "" {
			return note.ID, nil
		}
	}

	return 0, nil
}

func (s *server) ensureBoardExists(boardID int, name string) error {
	b, err := s.dbHandler.GetBoard(boardID)
	if err != nil {
		return fmt.Errorf("querying db for board: %w", err)
	}

	if b == nil {
		n := db.NewBoard(boardID, name)
		if err := s.dbHandler.UpsertBoard(n); err != nil {
			return fmt.Errorf("inserting new board into db: %w", err)
		}
	}

	return nil
}

func (s *server) ensureStatusExists(ctx context.Context, statusID, boardID int, boardName string) error {
	st, err := s.dbHandler.GetStatus(statusID)
	if err != nil {
		return fmt.Errorf("querying db for status: %w", err)
	}

	if st == nil {

		if boardID == 0 {
			if err := s.ensureBoardExists(boardID, boardName); err != nil {
				return fmt.Errorf("ensuring board exists for status: %w", err)
			}
		}

		r, err := s.cwClient.GetBoardStatus(ctx, boardID, statusID, nil)
		if err != nil {
			return checkCWError("getting status", "status", err, statusID)
		}

		n := db.NewStatus(statusID, boardID, r.Name, r.ClosedStatus, !r.Inactive)
		if err := s.dbHandler.UpsertStatus(n); err != nil {
			return fmt.Errorf("inserting new status into db: %w", err)
		}
	}

	return nil
}

func (s *server) ensureContactExists(ctx context.Context, contactID int) error {
	c, err := s.dbHandler.GetContact(contactID)
	if err != nil {
		return fmt.Errorf("querying db for contact: %w", err)
	}

	if c == nil {
		r, err := s.cwClient.GetContact(ctx, contactID, nil)
		if err != nil {
			return checkCWError("getting contact", "contact", err, contactID)
		}

		if r.Company.ID != 0 {
			if err := s.ensureCompanyExists(r.Company.ID, r.Company.Name); err != nil {
				return fmt.Errorf("ensuring company exists for contact: %w", err)
			}
		}

		n := db.NewContact(contactID, r.FirstName, r.LastName, r.Company.ID)
		if err := s.dbHandler.UpsertContact(n); err != nil {
			return fmt.Errorf("inserting new contact into db: %w", err)
		}
	}

	return nil
}

func (s *server) ensureCompanyExists(companyID int, name string) error {
	c, err := s.dbHandler.GetCompany(companyID)
	if err != nil {
		return fmt.Errorf("querying db for company: %w", err)
	}

	if c == nil {
		n := db.NewCompany(companyID, name)
		if err := s.dbHandler.UpsertCompany(n); err != nil {
			return fmt.Errorf("inserting new company into db: %w", err)
		}
	}

	return nil
}

func (s *server) ensureTicketNoteExists(ctx context.Context, ticketID, noteID int) error {
	note, err := s.dbHandler.GetTicketNote(noteID)
	if err != nil {
		return fmt.Errorf("querying db for note: %w", err)
	}

	if note == nil {
		r, err := s.cwClient.GetServiceTicketNote(ctx, ticketID, noteID, nil)
		if err != nil {
			return checkCWError("getting ticket note", "ticket", err, noteID)
		}

		if r.Contact.ID != 0 {
			if err := s.ensureContactExists(ctx, r.Contact.ID); err != nil {
				return fmt.Errorf("ensuring contact exists for ticket note: %w", err)
			}
		}

		// TODO: check if member exists, if not, create it

		n := db.NewTicketNote(ticketID, noteID, r.Contact.ID, r.Member.ID, r.Text, r.DateCreated, r.InternalAnalysisFlag)
		if err := s.dbHandler.UpsertTicketNote(n); err != nil {
			return fmt.Errorf("inserting new ticket note into db: %w", err)
		}
	}

	return nil
}
