package cwsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

var ErrTicketWasDeleted = errors.New("ticket was deleted from connectwise")

type Request struct {
	*models.FullTicket
	NoProcReason string
	cd           CWData
}

type CWData struct {
	ticket *psa.Ticket
	note   *psa.ServiceTicketNote
}

func (s *Service) DeleteTicket(ctx context.Context, id int) error {
	if err := s.Tickets.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting ticket from store: %w", err)
	}
	return nil
}

func (s *Service) ProcessTicket(ctx context.Context, id int, caller string) (*models.FullTicket, error) {
	req, err := s.processTicket(ctx, id, caller)
	if err != nil {
		return nil, err
	}

	return req.FullTicket, nil
}

func (s *Service) processTicket(ctx context.Context, id int, caller string) (req *Request, err error) {
	// TODO: make this less bad
	req = &Request{
		NoProcReason: "",
		cd:           CWData{},
	}

	logger := slog.Default()
	defer func() {
		logRequest(req, err, logger)
	}()

	cd, err := s.getCwData(id)
	if err != nil {
		if errors.Is(err, ErrTicketWasDeleted) {
			req.NoProcReason = "ticket was deleted from connectwise"
			return req, nil
		}

		return req, fmt.Errorf("getting ticket data from connectwise: %w", err)
	}

	if cd.ticket == nil {
		return req, fmt.Errorf("no data returned from connectwise for ticket %d", id)
	}

	// TODO: this is a bandaid. Move this logic to the repo.
	txSvc := s
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return req, fmt.Errorf("beginning tx: %w", err)
		}

		txSvc = s.WithTX(tx)

		defer func() {
			_ = tx.Rollback(ctx)
		}()
	}

	cwt := cd.ticket
	logger = logger.With(slog.Int("ticket_id", cwt.ID), slog.String("caller", caller))

	board, err := txSvc.ensureBoard(ctx, cwt.Board.ID)
	if err != nil {
		return req, fmt.Errorf("ensuring board in store: %w", err)
	}
	logger = logger.With(boardLogGrp(board))

	company, err := txSvc.ensureCompany(ctx, cwt.Company.ID)
	if err != nil {
		return nil, fmt.Errorf("ensuring company in store: %w", err)
	}
	logger = logger.With(companyLogGrp(company))

	var contact *models.Contact
	if cwt.Contact.ID != 0 {
		contact, err = txSvc.ensureContact(ctx, cwt.Contact.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring ticket contact in store: %w", err)
		}
		logger = logger.With(contactLogGrp(contact))
	}

	var owner *models.Member
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

	var rsc []*models.Member
	if ticket.Resources != nil && *ticket.Resources != "" {
		logger = logger.With(slog.String("resources", *ticket.Resources))
		ids := resourceStringToSlice(*ticket.Resources)
		for _, i := range ids {
			member, err := txSvc.ensureMemberByIdentifier(ctx, i)
			if err != nil {
				logger.Warn("cwsvc: error getting resource member by identifier", "identifier", i, "error", err)
				continue
			}

			rsc = append(rsc, member)
		}
	}

	var note *models.FullTicketNote
	if cd.note != nil && cd.note.ID != 0 {
		note, err = txSvc.ensureTicketNote(ctx, cd.note)
		if err != nil {
			return nil, fmt.Errorf("ensuring ticket note in store: %w", err)
		}
		logger = logger.With(noteLogGrp(note))
	}

	// TODO: this is a bandaid. Move this logic to the repo.
	if s.pool != nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("committing transaction: %w", err)
		}
	}

	req.FullTicket = &models.FullTicket{
		Board:      *board,
		Ticket:     *ticket,
		Company:    *company,
		Contact:    contact,
		Owner:      owner,
		LatestNote: note,
		Resources:  rsc,
	}

	return req, nil
}

func (s *Service) getCwData(ticketID int) (CWData, error) {
	t, err := s.CWClient.GetTicket(ticketID, nil)
	if err != nil {
		if errors.Is(err, psa.ErrNotFound) {
			return CWData{}, ErrTicketWasDeleted
		}
		return CWData{}, fmt.Errorf("getting ticket: %w", err)
	}

	n, err := s.CWClient.GetMostRecentTicketNote(ticketID)
	if err != nil && !errors.Is(err, psa.ErrNotFound) {
		return CWData{}, fmt.Errorf("getting most recent ticket note: %w", err)
	}

	return CWData{ticket: t, note: n}, nil
}

func (s *Service) ensureBoard(ctx context.Context, id int) (*models.Board, error) {
	b, err := s.Boards.Get(ctx, id)
	if err == nil {
		return b, nil
	}

	if !errors.Is(err, models.ErrBoardNotFound) {
		return nil, fmt.Errorf("getting board from store: %w", err)
	}

	cw, err := s.CWClient.GetBoard(id, nil)
	if err != nil {
		return nil, fmt.Errorf("getting board from cw: %w", err)
	}

	b, err = s.Boards.Upsert(ctx, &models.Board{
		ID:   cw.ID,
		Name: cw.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting board into store: %w", err)
	}

	return b, nil
}

func (s *Service) ensureCompany(ctx context.Context, id int) (*models.Company, error) {
	c, err := s.Companies.Get(ctx, id)
	if err == nil {
		return c, nil
	}

	if !errors.Is(err, models.ErrCompanyNotFound) {
		return nil, fmt.Errorf("getting company from store: %w", err)
	}

	cw, err := s.CWClient.GetCompany(id, nil)
	if err != nil {
		return nil, fmt.Errorf("getting company from cw: %w", err)
	}

	c, err = s.Companies.Upsert(ctx, &models.Company{
		ID:   cw.Id,
		Name: cw.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting company into store: %w", err)
	}

	return c, nil
}

func (s *Service) ensureContact(ctx context.Context, id int) (*models.Contact, error) {
	c, err := s.Contacts.Get(ctx, id)
	if err == nil {
		return c, nil
	}

	if !errors.Is(err, models.ErrContactNotFound) {
		return nil, fmt.Errorf("getting contact from store: %w", err)
	}

	cw, err := s.CWClient.GetContact(id, nil)
	if err != nil {
		return nil, fmt.Errorf("getting contact from cw: %w", err)
	}

	var compID *int
	if cw.Company.ID != 0 {
		comp, err := s.ensureCompany(ctx, cw.Company.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring contact's company is in store: %w", err)
		}
		compID = intToPtr(comp.ID)
	}

	c, err = s.Contacts.Upsert(ctx, &models.Contact{
		ID:        cw.ID,
		FirstName: cw.FirstName,
		LastName:  strToPtr(cw.LastName),
		CompanyID: compID,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting contact into store: %w", err)
	}

	return c, nil
}

func (s *Service) ensureMemberByIdentifier(ctx context.Context, identifier string) (*models.Member, error) {
	m, err := s.Members.GetByIdentifier(ctx, identifier)
	if err == nil {
		return m, nil
	}

	if !errors.Is(err, models.ErrMemberNotFound) {
		return nil, fmt.Errorf("getting member from store: %w", err)
	}

	cw, err := s.CWClient.GetMemberByIdentifier(identifier)
	if err != nil {
		return nil, fmt.Errorf("getting member from cw by identifier: %w", err)
	}

	return s.ensureMember(ctx, cw.ID)
}

func (s *Service) ensureMember(ctx context.Context, id int) (*models.Member, error) {
	m, err := s.Members.Get(ctx, id)
	if err == nil {
		return m, nil
	}

	if !errors.Is(err, models.ErrMemberNotFound) {
		return nil, fmt.Errorf("getting member from store: %w", err)
	}

	cw, err := s.CWClient.GetMember(id, nil)
	if err != nil {
		return nil, fmt.Errorf("getting member from cw: %w", err)
	}

	m, err = s.Members.Upsert(ctx, &models.Member{
		ID:           cw.ID,
		Identifier:   cw.Identifier,
		FirstName:    cw.FirstName,
		LastName:     cw.LastName,
		PrimaryEmail: cw.PrimaryEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting member into store: %w", err)
	}

	return m, nil
}

func (s *Service) ensureTicket(ctx context.Context, cwt *psa.Ticket) (*models.Ticket, error) {
	if cwt == nil {
		return nil, errors.New("received nil ticket")
	}

	t, err := s.Tickets.Upsert(ctx, &models.Ticket{
		ID:        cwt.ID,
		Summary:   cwt.Summary,
		BoardID:   cwt.Board.ID,
		OwnerID:   intToPtr(cwt.Owner.ID),
		CompanyID: cwt.Company.ID,
		ContactID: intToPtr(cwt.Contact.ID),
		Resources: &cwt.Resources,
		UpdatedBy: &cwt.Info.UpdatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("upserting ticket: %w", err)
	}

	return t, nil
}

func (s *Service) ensureTicketNote(ctx context.Context, cwn *psa.ServiceTicketNote) (*models.FullTicketNote, error) {
	if cwn == nil {
		return nil, errors.New("received nil ticket note")
	}

	memberID, err := s.getNoteMemberID(ctx, cwn)
	if err != nil {
		return nil, fmt.Errorf("getting member data: %w", err)
	}

	contactID, err := s.getNoteContactID(ctx, cwn)
	if err != nil {
		return nil, fmt.Errorf("getting contact data: %w ", err)
	}

	n, err := s.Notes.Get(ctx, cwn.ID)
	if err == nil {
		return models.TicketNoteToFullTicketNote(ctx, n, s.Members, s.Contacts)
	}

	if !errors.Is(err, models.ErrTicketNoteNotFound) {
		return nil, fmt.Errorf("getting note from store: %w", err)
	}

	n, err = s.Notes.Upsert(ctx, &models.TicketNote{
		ID:        cwn.ID,
		TicketID:  cwn.TicketId,
		Content:   strToPtr(cwn.Text),
		MemberID:  memberID,
		ContactID: contactID,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting note into store: %w", err)
	}

	return models.TicketNoteToFullTicketNote(ctx, n, s.Members, s.Contacts)
}

func (s *Service) getNoteContactID(ctx context.Context, n *psa.ServiceTicketNote) (*int, error) {
	if n.Contact.ID != 0 {
		c, err := s.ensureContact(ctx, n.Contact.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring contact in store: %w", err)
		}

		return intToPtr(c.ID), nil
	}

	return nil, nil
}

func (s *Service) getNoteMemberID(ctx context.Context, n *psa.ServiceTicketNote) (*int, error) {
	if n.Member.ID != 0 {
		c, err := s.ensureMember(ctx, n.Member.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring member in store: %w", err)
		}

		return intToPtr(c.ID), nil
	}

	return nil, nil
}

func resourceStringToSlice(s string) []string {
	rsc := strings.Split(s, ",")
	for i, r := range rsc {
		rsc[i] = strings.TrimSpace(r)
	}

	return rsc
}

func intToPtr(i int) *int {
	if i == 0 {
		return nil
	}
	val := i
	return &val
}

func strToPtr(s string) *string {
	if s == "" {
		return nil
	}
	val := s
	return &val
}

func logRequest(req *Request, err error, logger *slog.Logger) {
	if req.NoProcReason != "" {
		logger = logger.With("no_process_reason", req.NoProcReason)
	}

	if err != nil {
		logger.Error("error occured processing ticket", "error", err)
	} else {
		logger.Info("ticket processed")
	}
}

func companyLogGrp(company *models.Company) slog.Attr {
	return slog.Group("company", "id", company.ID, "name", company.Name)
}

func boardLogGrp(board *models.Board) slog.Attr {
	return slog.Group("board", "id", board.ID, "name", board.Name)
}

func ownerLogGrp(owner *models.Member) slog.Attr {
	return slog.Group("owner",
		"id", owner.ID,
		"identifier", owner.Identifier,
		"first_name", owner.FirstName,
		"last_name", owner.LastName,
	)
}

func contactLogGrp(contact *models.Contact) slog.Attr {
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

func noteLogGrp(note *models.FullTicketNote) slog.Attr {
	var (
		senderID   int
		senderType string
	)

	if note.Member != nil {
		senderID = note.Member.ID
		senderType = "member"
	} else if note.Contact != nil {
		senderID = note.Contact.ID
		senderType = "contact"
	}

	return slog.Group("latest_note",
		"id", note.ID,
		"sender_id", senderID,
		"sender_type", senderType,
	)
}
