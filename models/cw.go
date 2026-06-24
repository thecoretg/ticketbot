package models

import (
	"errors"
	"time"
)

var ErrBoardNotFound = errors.New("board not found")

type Board struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}

var ErrCompanyNotFound = errors.New("company not found")

type Company struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
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

var ErrTicketNotFound = errors.New("ticket not found")

type Ticket struct {
	ID        int       `json:"id"`
	Summary   string    `json:"summary"`
	BoardID   int       `json:"board_id"`
	StatusID  int       `json:"status_id"`
	OwnerID   *int      `json:"owner_id"`
	CompanyID int       `json:"company_id"`
	ContactID *int      `json:"contact_id"`
	TypeID    *int      `json:"type_id"`
	SubTypeID *int      `json:"subtype_id"`
	ItemID    *int      `json:"item_id"`
	Resources *string   `json:"resources"`
	UpdatedBy *string   `json:"updated_by"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}

type FullTicket struct {
	Ticket     Ticket
	Board      Board
	Status     TicketStatus
	Company    Company
	Contact    *Contact
	Owner      *Member
	Type       *TicketType
	SubType    *TicketSubType
	Item       *TicketItem
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

type NoteFlags struct {
	Discussion bool
	Internal   bool
	Resolution bool
}

type FullTicketNote struct {
	TicketNote
	Member  *Member
	Contact *Contact
	Flags   *NoteFlags // ephemeral; populated from live CW data during processing, not stored in DB
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

var ErrTicketTypeNotFound = errors.New("ticket type not found")

type TicketType struct {
	ID          int       `json:"id"`
	BoardID     int       `json:"board_id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	DefaultFlag bool      `json:"default_flag"`
	Inactive    bool      `json:"inactive"`
	UpdatedOn   time.Time `json:"updated_on"`
	AddedOn     time.Time `json:"added_on"`
	Deleted     bool      `json:"deleted"`
}

var ErrTicketSubTypeNotFound = errors.New("ticket subtype not found")

// TicketSubType is a board-scoped subtype. TypeAssociationIDs records which Types
// the subtype belongs to (a subtype may be shared across multiple types).
type TicketSubType struct {
	ID                 int       `json:"id"`
	BoardID            int       `json:"board_id"`
	Name               string    `json:"name"`
	Inactive           bool      `json:"inactive"`
	TypeAssociationIDs []int     `json:"type_association_ids"`
	UpdatedOn          time.Time `json:"updated_on"`
	AddedOn            time.Time `json:"added_on"`
	Deleted            bool      `json:"deleted"`
}

var ErrTicketItemNotFound = errors.New("ticket item not found")

type TicketItem struct {
	ID        int       `json:"id"`
	BoardID   int       `json:"board_id"`
	Name      string    `json:"name"`
	Inactive  bool      `json:"inactive"`
	UpdatedOn time.Time `json:"updated_on"`
	AddedOn   time.Time `json:"added_on"`
	Deleted   bool      `json:"deleted"`
}
