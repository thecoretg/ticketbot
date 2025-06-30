package ticketbot

import (
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"time"
)

type DBHandler struct {
	db *sqlx.DB
}

var tablesStmt = `
CREATE TABLE IF NOT EXISTS board (
    board_id INT PRIMARY KEY,
    board_name VARCHAR(50) NOT NULL 
);

CREATE TABLE IF NOT EXISTS status (
    status_id INT PRIMARY KEY,
    status_name VARCHAR(50) NOT NULL UNIQUE,
    board_id INT NOT NULL REFERENCES board(board_id),
    closed BOOLEAN NOT NULL,
    active BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS company (
    company_id INT PRIMARY KEY,
    company_name VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS contact (
    contact_id INT PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50),
    company_id INT REFERENCES company(company_id)
);

CREATE TABLE IF NOT EXISTS member (
    member_id INT PRIMARY KEY,
    identifier VARCHAR(15) NOT NULL UNIQUE,
    first_name VARCHAR(30) NOT NULL,
    last_name VARCHAR(30) NOT NULL,
    email VARCHAR(50) NOT NULL,
    phone VARCHAR(10)
);

CREATE TABLE IF NOT EXISTS ticket (
    ticket_id INT PRIMARY KEY,
    board_id INT NOT NULL REFERENCES board(board_id),
    company_id INT NOT NULL REFERENCES company(company_id),
    contact_id INT REFERENCES contact(contact_id),
    summary VARCHAR(100) NOT NULL,
    latest_note_id INT,
    owner_id INT REFERENCES member(member_id),
    resources TEXT,
    created_on TIMESTAMP NOT NULL,
    updated_on TIMESTAMP NOT NULL,
    closed_on TIMESTAMP,
    closed BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS ticket_note (
    note_id INT PRIMARY KEY,
    ticket_id INT NOT NULL REFERENCES ticket(ticket_id),
    contact_id INT REFERENCES contact(contact_id),
    member_id INT REFERENCES member(member_id),
    created_on TIMESTAMP NOT NULL,
    internal BOOLEAN DEFAULT FALSE
);
`

type TicketNote struct {
	ID        int       `db:"note_id"`
	TicketID  int       `db:"ticket_id"`
	ContactID *int      `db:"contact_id"`
	MemberID  *int      `db:"member_id"`
	CreatedOn time.Time `db:"created_on"`
	Internal  bool      `db:"internal"`
}

type Board struct {
	ID   int    `db:"board_id"`
	Name string `db:"board_name"`
}

func NewBoard(id int, name string) *Board {
	return &Board{
		ID:   id,
		Name: name,
	}
}

type Status struct {
	ID      int    `db:"status_id"`
	Name    string `db:"status_name"`
	BoardID int    `db:"board_id"`
	Closed  bool   `db:"closed"`
	Active  bool   `db:"active"`
}

func NewStatus(id, boardID int, name string, closed, active bool) *Status {
	return &Status{
		ID:      id,
		Name:    name,
		BoardID: boardID,
		Closed:  closed,
		Active:  active,
	}
}

type Company struct {
	ID   int    `db:"company_id"`
	Name string `db:"company_name"`
}

func NewCompany(id int, name string) *Company {
	return &Company{
		ID:   id,
		Name: name,
	}
}

type Contact struct {
	ID        int     `db:"contact_id"`
	FirstName string  `db:"first_name"`
	LastName  *string `db:"last_name"`
	CompanyID *int    `db:"company_id"`
}

func NewContact(id int, firstName, lastName string, companyID int) *Contact {
	return &Contact{
		ID:        id,
		FirstName: firstName,
		LastName:  strToPtr(lastName),
		CompanyID: intToPtr(companyID),
	}
}

type Member struct {
	ID         int     `db:"member_id"`
	Identifier string  `db:"identifier"`
	FirstName  string  `db:"first_name"`
	LastName   string  `db:"last_name"`
	Email      string  `db:"email"`
	Phone      *string `db:"phone"`
}

func NewMember(id int, identifier, firstName, lastName, email, phone string) *Member {
	return &Member{
		ID:         id,
		Identifier: identifier,
		FirstName:  firstName,
		LastName:   lastName,
		Email:      email,
		Phone:      strToPtr(phone),
	}
}

type Ticket struct {
	ID         int        `db:"ticket_id"`
	Board      int        `db:"board_id"`
	Company    int        `db:"company_id"`
	Contact    *int       `db:"contact_id"`
	Summary    string     `db:"summary"`
	LatestNote *int       `db:"latest_note_id"`
	Owner      *int       `db:"owner_id"`
	Resources  *string    `db:"resources"`
	Created    time.Time  `db:"created_on"`
	Updated    time.Time  `db:"updated_on"`
	ClosedOn   *time.Time `db:"closed_on"`
	Closed     bool       `db:"closed"`
}

func NewTicket(ticketID, boardID, companyID, contactID, latestNoteID, ownerID int, summary, resources string, createdOn, updatedOn, closedOn time.Time, closed bool) *Ticket {
	return &Ticket{
		ID:         ticketID,
		Board:      boardID,
		Company:    companyID,
		Contact:    intToPtr(contactID),
		LatestNote: intToPtr(latestNoteID),
		Owner:      intToPtr(ownerID),
		Summary:    summary,
		Resources:  strToPtr(resources),
		Created:    createdOn,
		Updated:    updatedOn,
		ClosedOn:   timeToPtr(closedOn),
		Closed:     closed,
	}
}

func InitDB(connString string) (*DBHandler, error) {
	db, err := sqlx.Connect("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	db.MustExec(tablesStmt)

	return &DBHandler{
		db: db,
	}, nil
}

func (h *DBHandler) GetTicket(ticketID int) (*Ticket, error) {
	t := &Ticket{}
	err := h.db.Get(t, "SELECT * FROM ticket WHERE ticket_id = $1", ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting ticket by id: %w", err)
	}

	if t.ID == 0 {
		return nil, nil
	}

	return t, nil
}

func (h *DBHandler) ListTickets() ([]Ticket, error) {
	var tickets []Ticket
	if err := h.db.Select(&tickets, "SELECT * FROM ticket"); err != nil {
		return nil, fmt.Errorf("listing tickets: %w", err)
	}

	return tickets, nil
}

func (h *DBHandler) UpsertTicket(t *Ticket) error {
	_, err := h.db.NamedExec(upsertTicketSQL(), t)
	if err != nil {
		return fmt.Errorf("inserting ticket: %w", err)
	}
	return nil
}

func (h *DBHandler) DeleteTicket(ticketID int) error {
	_, err := h.db.Exec("DELETE FROM ticket WHERE ticket_id = $1", ticketID)
	if err != nil {
		return err
	}

	return nil
}

func (h *DBHandler) GetBoard(boardID int) (*Board, error) {
	b := &Board{}
	err := h.db.Get(b, "SELECT * FROM board WHERE board_id = $1", boardID)
	if err != nil {
		return nil, fmt.Errorf("getting board by id: %w", err)
	}
	if b.ID == 0 {
		return nil, nil
	}
	return b, nil
}

func (h *DBHandler) ListBoards() ([]Board, error) {
	var boards []Board
	if err := h.db.Select(&boards, "SELECT * FROM board"); err != nil {
		return nil, fmt.Errorf("listing boards: %w", err)
	}
	return boards, nil
}

func (h *DBHandler) UpsertBoard(b *Board) error {
	_, err := h.db.NamedExec(upsertBoardSQL(), b)
	if err != nil {
		return fmt.Errorf("inserting board: %w", err)
	}
	return nil
}

func (h *DBHandler) DeleteBoard(boardID int) error {
	_, err := h.db.Exec("DELETE FROM board WHERE board_id = $1", boardID)
	return err
}

func (h *DBHandler) GetStatus(statusID int) (*Status, error) {
	s := &Status{}
	err := h.db.Get(s, "SELECT * FROM status WHERE status_id = $1", statusID)
	if err != nil {
		return nil, fmt.Errorf("getting status by id: %w", err)
	}
	if s.ID == 0 {
		return nil, nil
	}
	return s, nil
}

func (h *DBHandler) ListStatuses() ([]Status, error) {
	var statuses []Status
	if err := h.db.Select(&statuses, "SELECT * FROM status"); err != nil {
		return nil, fmt.Errorf("listing statuses: %w", err)
	}
	return statuses, nil
}

func (h *DBHandler) UpsertStatus(s *Status) error {
	_, err := h.db.NamedExec(upsertStatusSQL(), s)
	if err != nil {
		return fmt.Errorf("inserting status: %w", err)
	}
	return nil
}

func (h *DBHandler) DeleteStatus(statusID int) error {
	_, err := h.db.Exec("DELETE FROM status WHERE status_id = $1", statusID)
	return err
}

func (h *DBHandler) GetCompany(companyID int) (*Company, error) {
	c := &Company{}
	err := h.db.Get(c, "SELECT * FROM company WHERE company_id = $1", companyID)
	if err != nil {
		return nil, fmt.Errorf("getting company by id: %w", err)
	}
	if c.ID == 0 {
		return nil, nil
	}
	return c, nil
}

func (h *DBHandler) ListCompanies() ([]Company, error) {
	var companies []Company
	if err := h.db.Select(&companies, "SELECT * FROM company"); err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	return companies, nil
}

func (h *DBHandler) UpsertCompany(c *Company) error {
	_, err := h.db.NamedExec(upsertCompanySQL(), c)
	if err != nil {
		return fmt.Errorf("inserting company: %w", err)
	}
	return nil
}

func (h *DBHandler) DeleteCompany(companyID int) error {
	_, err := h.db.Exec("DELETE FROM company WHERE company_id = $1", companyID)
	return err
}

func (h *DBHandler) GetContact(contactID int) (*Contact, error) {
	c := &Contact{}
	err := h.db.Get(c, "SELECT * FROM contact WHERE contact_id = $1", contactID)
	if err != nil {
		return nil, fmt.Errorf("getting contact by id: %w", err)
	}
	if c.ID == 0 {
		return nil, nil
	}
	return c, nil
}

func (h *DBHandler) ListContacts() ([]Contact, error) {
	var contacts []Contact
	if err := h.db.Select(&contacts, "SELECT * FROM contact"); err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}
	return contacts, nil
}

func (h *DBHandler) UpsertContact(c *Contact) error {
	_, err := h.db.NamedExec(upsertContactSQL(), c)
	if err != nil {
		return fmt.Errorf("inserting contact: %w", err)
	}
	return nil
}

func (h *DBHandler) DeleteContact(contactID int) error {
	_, err := h.db.Exec("DELETE FROM contact WHERE contact_id = $1", contactID)
	return err
}

func (h *DBHandler) GetMember(memberID int) (*Member, error) {
	m := &Member{}
	err := h.db.Get(m, "SELECT * FROM member WHERE member_id = $1", memberID)
	if err != nil {
		return nil, fmt.Errorf("getting member by id: %w", err)
	}
	if m.ID == 0 {
		return nil, nil
	}
	return m, nil
}

func (h *DBHandler) ListMembers() ([]Member, error) {
	var members []Member
	if err := h.db.Select(&members, "SELECT * FROM member"); err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	return members, nil
}

func (h *DBHandler) UpsertMember(m *Member) error {
	_, err := h.db.NamedExec(upsertMemberSQL(), m)
	if err != nil {
		return fmt.Errorf("inserting member: %w", err)
	}
	return nil
}

func (h *DBHandler) DeleteMember(memberID int) error {
	_, err := h.db.Exec("DELETE FROM member WHERE member_id = $1", memberID)
	return err
}

func upsertMemberSQL() string {
	return `INSERT INTO member (member_id, identifier, first_name, last_name, email, phone)
		VALUES (:member_id, :identifier, :first_name, :last_name, :email, :phone)
		ON CONFLICT (member_id) DO UPDATE SET
			identifier = EXCLUDED.identifier,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone`
}

func upsertContactSQL() string {
	return `INSERT INTO contact (contact_id, first_name, last_name, company_id)
		VALUES (:contact_id, :first_name, :last_name, :company_id)
		ON CONFLICT (contact_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			company_id = EXCLUDED.company_id`
}

func upsertTicketSQL() string {
	return `INSERT INTO ticket (ticket_id, board_id, company_id, contact_id, summary, latest_note_id, owner_id, resources, created_on, updated_on, closed_on, closed)
		VALUES (:ID, :Board, :Company, :Contact, :Summary, :LatestNote, :Owner, :Resources, :Created, :Updated, :ClosedOn, :Closed)
		ON CONFLICT (ticket_id) DO UPDATE SET
			board_id = EXCLUDED.board_id,
			company_id = EXCLUDED.company_id,
			contact_id = EXCLUDED.contact_id,
			summary = EXCLUDED.summary,
			latest_note_id = EXCLUDED.latest_note_id,
			owner_id = EXCLUDED.owner_id,
			resources = EXCLUDED.resources,
			created_on = EXCLUDED.created_on,
			updated_on = EXCLUDED.updated_on,
			closed_on = EXCLUDED.closed_on,
			closed = EXCLUDED.closed`
}

func upsertBoardSQL() string {
	return `INSERT INTO board (board_id, board_name)
		VALUES (:board_id, :board_name)
		ON CONFLICT (board_id) DO UPDATE SET
			board_name = EXCLUDED.board_name`
}

func upsertStatusSQL() string {
	return `INSERT INTO status (status_id, status_name, board_id, closed, active)
		VALUES (:status_id, :status_name, :board_id, :closed, :active)
		ON CONFLICT (status_id) DO UPDATE SET
			status_name = EXCLUDED.status_name,
			board_id = EXCLUDED.board_id,
			closed = EXCLUDED.closed,
			active = EXCLUDED.active`
}

func upsertCompanySQL() string {
	return `INSERT INTO company (company_id, company_name)
		VALUES (:company_id, :company_name)
		ON CONFLICT (company_id) DO UPDATE SET
			company_name = EXCLUDED.company_name`
}
