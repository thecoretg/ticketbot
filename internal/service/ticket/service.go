package ticket

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Boards    models.BoardRepository
	Companies models.CompanyRepository
	Contacts  models.ContactRepository
	Members   models.MemberRepository
	Tickets   models.TicketRepository
	Notes     models.TicketNoteRepository

	pool        *pgxpool.Pool
	cwClient    *psa.Client
	ticketLocks *sync.Map
}

type FullTicket struct {
	Board   models.Board
	Ticket  models.Ticket
	Company models.Company
	Contact models.Contact
	Owner   models.Member
	Note    models.TicketNote
}

type cwData struct {
	ticket *psa.Ticket
	note   *psa.ServiceTicketNote
}

func New(pool *pgxpool.Pool, b models.BoardRepository, comp models.CompanyRepository, cn models.ContactRepository,
	mem models.MemberRepository, tix models.TicketRepository, nt models.TicketNoteRepository, cl *psa.Client) *Service {
	return &Service{
		Boards:    b,
		Companies: comp,
		Contacts:  cn,
		Members:   mem,
		Tickets:   tix,
		Notes:     nt,
		pool:      pool,
		cwClient:  cl,
	}
}

func (s *Service) withTx(tx pgx.Tx) *Service {
	return &Service{
		Boards:      s.Boards.WithTx(tx),
		Companies:   s.Companies.WithTx(tx),
		Contacts:    s.Contacts.WithTx(tx),
		Members:     s.Members.WithTx(tx),
		Tickets:     s.Tickets.WithTx(tx),
		Notes:       s.Notes.WithTx(tx),
		pool:        s.pool,
		cwClient:    s.cwClient,
		ticketLocks: s.ticketLocks,
	}
}

func (s *Service) Run(ctx context.Context, id int) (*FullTicket, error) {
	lock := s.getTicketLock(id)
	if !lock.TryLock() {
		lock.Lock()
	}

	defer func() {
		lock.Unlock()
	}()

	cd, err := s.getCwData(id)
	if err != nil {
		return nil, fmt.Errorf("getting ticket data from connectwise: %w", err)
	}

	if cd.ticket == nil {
		return nil, fmt.Errorf("no data returned from connectwise for ticket %d", id)
	}

	ft, err := s.processTicket(ctx, cd)
	if err != nil {
		return nil, fmt.Errorf("ensuring all data for ticket: %w", err)
	}

	return ft, nil
}

func (s *Service) processTicket(ctx context.Context, cd cwData) (*FullTicket, error) {
	logger := slog.Default()
	defer func() { logger.Info("ticket processed") }()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}

	txSvc := s.withTx(tx)

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	cwt := cd.ticket
	logger = logger.With(slog.Int("ticket_id", cwt.ID))

	board, err := txSvc.ensureBoard(ctx, cwt.Board.ID)
	if err != nil {
		return nil, fmt.Errorf("ensuring board in store: %w", err)
	}
	logger = logger.With(boardLogGrp(board))

	company, err := txSvc.ensureCompany(ctx, cwt.Company.ID)
	if err != nil {
		return nil, fmt.Errorf("ensuring company in store: %w", err)
	}
	logger = logger.With(companyLogGrp(company))

	contact := models.Contact{}
	if cwt.Contact.ID != 0 {
		contact, err = txSvc.ensureContact(ctx, cwt.Contact.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring ticket contact in store: %w", err)
		}
		logger = logger.With(contactLogGrp(contact))
	}

	owner := models.Member{}
	if cwt.Owner.ID != 0 {
		owner, err = txSvc.ensureMember(ctx, cwt.Owner.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring ticket owner in store: %w", err)
		}
		logger = logger.With(ownerLogGrp(owner))
	}

	ticket, err := txSvc.ensureTicket(ctx, cd.ticket)
	if err != nil {
		return nil, fmt.Errorf("ensuring ticket in store: %w", err)
	}

	note := models.TicketNote{}
	if cd.note != nil && cd.note.ID != 0 {
		note, err = txSvc.ensureTicketNote(ctx, cd.note)
		if err != nil {
			return nil, fmt.Errorf("ensuring ticket note in store: %w", err)
		}
		logger = logger.With(noteLogGrp(note))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return &FullTicket{
		Board:   board,
		Ticket:  ticket,
		Company: company,
		Contact: contact,
		Owner:   owner,
		Note:    note,
	}, nil
}

func companyLogGrp(company models.Company) slog.Attr {
	return slog.Group("company", "id", company.ID, "name", company.Name)
}

func boardLogGrp(board models.Board) slog.Attr {
	return slog.Group("board", "id", board.ID, "name", board.Name)
}

func ownerLogGrp(owner models.Member) slog.Attr {
	return slog.Group("owner",
		"id", owner.ID,
		"identifier", owner.Identifier,
		"first_name", owner.FirstName,
		"last_name", owner.LastName,
	)
}

func contactLogGrp(contact models.Contact) slog.Attr {
	ln := "None"
	if contact.LastName != nil {
		ln = *contact.LastName
	}

	return slog.Group("contact",
		"id", contact.ID,
		"first_name", contact.FirstName,
		"last_name", ln,
	)
}

func noteLogGrp(note models.TicketNote) slog.Attr {
	var (
		senderID   int
		senderType string
	)

	if note.MemberID != nil {
		senderID = *note.MemberID
		senderType = "member"
	} else if note.ContactID != nil {
		senderID = *note.ContactID
		senderType = "contact"
	}

	return slog.Group("latest_note",
		"id", note.ID,
		"sender_id", senderID,
		"sender_type", senderType,
	)
}

func (s *Service) getTicketLock(id int) *sync.Mutex {
	li, _ := s.ticketLocks.LoadOrStore(id, &sync.Mutex{})
	return li.(*sync.Mutex)
}
