package models

import (
	"encoding/json"
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
	ID           int       `json:"id"`
	Summary      string    `json:"summary"`
	BoardID      int       `json:"board_id"`
	StatusID     int       `json:"status_id"`
	OwnerID      *int      `json:"owner_id"`
	CompanyID    int       `json:"company_id"`
	ContactID    *int      `json:"contact_id"`
	Resources    *string   `json:"resources"`
	UpdatedBy    *string   `json:"updated_by"`
	PriorityID   *int      `json:"priority_id"`
	PriorityName *string   `json:"priority_name"`
	TypeID       *int      `json:"type_id"`
	TypeName     *string   `json:"type_name"`
	SubTypeID    *int      `json:"subtype_id"`
	SubTypeName  *string   `json:"subtype_name"`
	ItemID       *int      `json:"item_id"`
	ItemName     *string   `json:"item_name"`
	ClosedFlag   bool      `json:"closed_flag"`
	LatestNoteID *int      `json:"latest_note_id"`
	UpdatedOn    time.Time `json:"updated_on"`
	AddedOn      time.Time `json:"added_on"`
	Deleted      bool      `json:"deleted"`

	// Raw is the full ConnectWise ticket JSON as last fetched. It is excluded from API output;
	// use the dedicated raw endpoint instead.
	Raw json.RawMessage `json:"-"`
}

// TicketListItem is a ticket row joined with the display names the dashboard needs.
type TicketListItem struct {
	Ticket
	BoardName   string `json:"board_name"`
	StatusName  string `json:"status_name"`
	CompanyName string `json:"company_name"`
	OwnerName   string `json:"owner_name"`
	CWURL       string `json:"cw_url"`
}

// TicketFilter narrows a paged ticket listing. Nil pointers mean "no filter".
type TicketFilter struct {
	BoardID        *int
	StatusID       *int
	Closed         *bool
	IncludeDeleted bool
	Search         string
	Page           int
	PageSize       int
}

type TicketPage struct {
	Items    []*TicketListItem `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// TicketDetail is everything the dashboard shows for a single ticket.
type TicketDetail struct {
	Ticket     TicketListItem  `json:"ticket"`
	Board      Board           `json:"board"`
	Status     TicketStatus    `json:"status"`
	Company    Company         `json:"company"`
	Contact    *Contact        `json:"contact"`
	Owner      *Member         `json:"owner"`
	Resources  []*Member       `json:"resources"`
	LatestNote *FullTicketNote `json:"latest_note"`
	Events     []*TicketEvent  `json:"events"`
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
	ID                   int       `json:"id"`
	TicketID             int       `json:"ticket_id"`
	MemberID             *int      `json:"member_id"`
	ContactID            *int      `json:"contact_id"`
	Content              *string   `json:"text"`
	InternalAnalysisFlag bool      `json:"internal_analysis_flag"`
	UpdatedOn            time.Time `json:"updated_on"`
	AddedOn              time.Time `json:"added_on"`
	Deleted              bool      `json:"deleted"`
}

type FullTicketNote struct {
	TicketNote
	Member  *Member
	Contact *Contact
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
