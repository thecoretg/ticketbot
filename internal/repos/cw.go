package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type BoardRepository interface {
	WithTx(tx pgx.Tx) BoardRepository
	List(ctx context.Context) ([]*models.Board, error)
	Get(ctx context.Context, id int) (*models.Board, error)
	Upsert(ctx context.Context, b *models.Board) (*models.Board, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type CompanyRepository interface {
	WithTx(tx pgx.Tx) CompanyRepository
	List(ctx context.Context) ([]*models.Company, error)
	Get(ctx context.Context, id int) (*models.Company, error)
	Upsert(ctx context.Context, c *models.Company) (*models.Company, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type ContactRepository interface {
	WithTx(tx pgx.Tx) ContactRepository
	List(ctx context.Context) ([]*models.Contact, error)
	Get(ctx context.Context, id int) (*models.Contact, error)
	Upsert(ctx context.Context, c *models.Contact) (*models.Contact, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type MemberRepository interface {
	WithTx(tx pgx.Tx) MemberRepository
	List(ctx context.Context) ([]*models.Member, error)
	Get(ctx context.Context, id int) (*models.Member, error)
	GetByIdentifier(ctx context.Context, identifier string) (*models.Member, error)
	Upsert(ctx context.Context, c *models.Member) (*models.Member, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type TicketRepository interface {
	WithTx(tx pgx.Tx) TicketRepository
	List(ctx context.Context) ([]*models.Ticket, error)
	Get(ctx context.Context, id int) (*models.Ticket, error)
	Exists(ctx context.Context, id int) (bool, error)
	Upsert(ctx context.Context, c *models.Ticket) (*models.Ticket, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type TicketNoteRepository interface {
	WithTx(tx pgx.Tx) TicketNoteRepository
	ListByTicketID(ctx context.Context, ticketID int) ([]*models.TicketNote, error)
	ListAll(ctx context.Context) ([]*models.TicketNote, error)
	Get(ctx context.Context, id int) (*models.TicketNote, error)
	Upsert(ctx context.Context, c *models.TicketNote) (*models.TicketNote, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type TicketStatusRepository interface {
	WithTx(tx pgx.Tx) TicketStatusRepository
	List(ctx context.Context) ([]*models.TicketStatus, error)
	ListByBoard(ctx context.Context, boardID int) ([]*models.TicketStatus, error)
	Get(ctx context.Context, id int) (*models.TicketStatus, error)
	Upsert(ctx context.Context, s *models.TicketStatus) (*models.TicketStatus, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

func TicketNoteToFullTicketNote(ctx context.Context, note *models.TicketNote, m MemberRepository, c ContactRepository) (*models.FullTicketNote, error) {
	var (
		member  *models.Member
		contact *models.Contact
		err     error
	)

	if note.MemberID != nil {
		member, err = m.Get(ctx, *note.MemberID)
		if err != nil {
			return nil, err
		}
	}

	if note.ContactID != nil {
		contact, err = c.Get(ctx, *note.ContactID)
		if err != nil {
			return nil, err
		}
	}

	return &models.FullTicketNote{
		TicketNote: *note,
		Member:     member,
		Contact:    contact,
	}, nil
}
