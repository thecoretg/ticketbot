package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrBoardNotFound = errors.New("board not found")

type Board struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}

type BoardRepository interface {
	WithTx(tx pgx.Tx) BoardRepository
	List(ctx context.Context) ([]*Board, error)
	Get(ctx context.Context, id int) (*Board, error)
	Upsert(ctx context.Context, b *Board) (*Board, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

var ErrCompanyNotFound = errors.New("company not found")

type Company struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}

type CompanyRepository interface {
	WithTx(tx pgx.Tx) CompanyRepository
	List(ctx context.Context) ([]*Company, error)
	Get(ctx context.Context, id int) (*Company, error)
	Upsert(ctx context.Context, c *Company) (*Company, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

var ErrContactNotFound = errors.New("contact not found")

type Contact struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  *string   `json:"last_name"`
	CompanyID *int      `json:"company_id"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}

type ContactRepository interface {
	WithTx(tx pgx.Tx) ContactRepository
	List(ctx context.Context) ([]*Contact, error)
	Get(ctx context.Context, id int) (*Contact, error)
	Upsert(ctx context.Context, c *Contact) (*Contact, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

var ErrMemberNotFound = errors.New("member not found")

type Member struct {
	ID           int       `json:"id"`
	Identifier   string    `json:"identifier"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PrimaryEmail string    `json:"primary_email"`
	UpdatedOn    time.Time `json:"updated_on"`
	AddedOn      time.Time `json:"added_on"`
	Deleted      bool      `json:"deleted"`
}

type MemberRepository interface {
	WithTx(tx pgx.Tx) MemberRepository
	List(ctx context.Context) ([]*Member, error)
	Get(ctx context.Context, id int) (*Member, error)
	GetByIdentifier(ctx context.Context, identifier string) (*Member, error)
	Upsert(ctx context.Context, c *Member) (*Member, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

var ErrTicketNotFound = errors.New("ticket not found")

type Ticket struct {
	ID        int       `json:"id"`
	Summary   string    `json:"summary"`
	BoardID   int       `json:"board_id"`
	StatusID  int       `json:"status_id"`
	OwnerID   *int      `json:"owner_id"`
	CompanyID int       `json:"company_id"`
	ContactID *int      `json:"contact_id"`
	Resources *string   `json:"resources"`
	UpdatedBy *string   `json:"updated_by"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}

type TicketRepository interface {
	WithTx(tx pgx.Tx) TicketRepository
	List(ctx context.Context) ([]*Ticket, error)
	Get(ctx context.Context, id int) (*Ticket, error)
	Exists(ctx context.Context, id int) (bool, error)
	Upsert(ctx context.Context, c *Ticket) (*Ticket, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type FullTicket struct {
	Ticket     Ticket
	Board      Board
	Status     TicketStatus
	Company    Company
	Contact    *Contact
	Owner      *Member
	LatestNote *FullTicketNote
	Resources  []*Member
}

var ErrTicketNoteNotFound = errors.New("ticket note not found")

type TicketNote struct {
	ID        int       `json:"id"`
	TicketID  int       `json:"ticket_id"`
	MemberID  *int      `json:"member_id"`
	ContactID *int      `json:"contact_id"`
	Content   *string   `json:"text"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}

type TicketNoteRepository interface {
	WithTx(tx pgx.Tx) TicketNoteRepository
	ListByTicketID(ctx context.Context, ticketID int) ([]*TicketNote, error)
	ListAll(ctx context.Context) ([]*TicketNote, error)
	Get(ctx context.Context, id int) (*TicketNote, error)
	Upsert(ctx context.Context, c *TicketNote) (*TicketNote, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type FullTicketNote struct {
	TicketNote
	Member  *Member
	Contact *Contact
}

func TicketNoteToFullTicketNote(ctx context.Context, note *TicketNote, m MemberRepository, c ContactRepository) (*FullTicketNote, error) {
	var (
		member  *Member
		contact *Contact
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

	return &FullTicketNote{
		TicketNote: *note,
		Member:     member,
		Contact:    contact,
	}, nil
}

var ErrTicketStatusNotFound = errors.New("ticket status not found")

type TicketStatus struct {
	ID             int       `json:"id"`
	BoardID        int       `json:"board_id"`
	Name           string    `json:"name"`
	DefaultStatus  bool      `json:"default_status"`
	DisplayOnBoard bool      `json:"display_on_board"`
	Inactive       bool      `json:"inactive"`
	Closed         bool      `json:"closed"`
	UpdatedOn      time.Time `json:"updated_on"`
	AddedOn        time.Time `json:"added_on"`
	Deleted        bool      `json:"deleted"`
}

type TicketStatusRepository interface {
	WithTx(tx pgx.Tx) TicketStatusRepository
	List(ctx context.Context) ([]*TicketStatus, error)
	ListByBoard(ctx context.Context, boardID int) ([]*TicketStatus, error)
	Get(ctx context.Context, id int) (*TicketStatus, error)
	Upsert(ctx context.Context, s *TicketStatus) (*TicketStatus, error)
	SoftDelete(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}
