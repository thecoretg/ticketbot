package db

import (
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	DB *sqlx.DB
}

var tablesStmt = `

CREATE TABLE IF NOT EXISTS webex_space (
	space_id TEXT PRIMARY KEY,
	space_name VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS board (
    board_id INT PRIMARY KEY,
    board_name VARCHAR(50) NOT NULL,
	notify_enabled BOOLEAN NOT NULL DEFAULT FALSE,	                                 
	webex_space_id TEXT REFERENCES webex_space(space_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS status (
    status_id INT PRIMARY KEY,
    status_name VARCHAR(50) NOT NULL,
    board_id INT NOT NULL REFERENCES board(board_id) ON DELETE SET NULL,
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
    company_id INT REFERENCES company(company_id) ON DELETE SET NULL
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
    status_id INT NOT NULL REFERENCES status(status_id),
    company_id INT NOT NULL REFERENCES company(company_id),
    contact_id INT REFERENCES contact(contact_id) ON DELETE SET NULL,
    summary VARCHAR(100) NOT NULL,
    latest_note_id INT,
    owner_id INT REFERENCES member(member_id) ON DELETE SET NULL,
    resources TEXT,
    created_on TIMESTAMP NOT NULL,
    updated_on TIMESTAMP NOT NULL,
    closed_on TIMESTAMP,
    closed BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS ticket_note (
    note_id INT PRIMARY KEY,
    ticket_id INT NOT NULL REFERENCES ticket(ticket_id) ON DELETE CASCADE,
    contact_id INT REFERENCES contact(contact_id) ON DELETE SET NULL,
    member_id INT REFERENCES member(member_id) ON DELETE SET NULL,
    content TEXT DEFAULT NULL,
    created_on TIMESTAMP NOT NULL,
    internal BOOLEAN DEFAULT FALSE,
   	notified BOOLEAN DEFAULT FALSE
);

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conname = 'fk_latest_note'
	) THEN
		ALTER TABLE ticket
		ADD CONSTRAINT fk_latest_note
		FOREIGN KEY (latest_note_id) REFERENCES ticket_note(note_id);
	END IF;
END $$
`

func InitDB(connString string) (*Handler, error) {
	db, err := sqlx.Connect("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	db.MustExec(tablesStmt)

	return &Handler{
		DB: db,
	}, nil
}
