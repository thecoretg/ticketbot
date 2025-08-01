package ticketbot

import (
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresStore struct {
	db *gorm.DB
}

func NewPostgresStore(conn string) (*PostgresStore, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: conn,
	}), &gorm.Config{})

	if err != nil {
		return nil, fmt.Errorf("opening postgres db: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("creating tables: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func createTables(db *gorm.DB) error {
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		return fmt.Errorf("automigrate ticket table: %w", err)
	}

	if err := db.AutoMigrate(&Board{}); err != nil {
		return fmt.Errorf("automigrate board table: %w", err)
	}

	//if err := db.AutoMigrate(&WebexRoom{}); err != nil {
	//	return fmt.Errorf("automigrate webex room table: %w", err)
	//}

	return nil
}

func (p *PostgresStore) UpsertTicket(ticket *Ticket) error {
	result := p.db.Save(ticket)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (p *PostgresStore) GetTicket(ticketID int) (*Ticket, error) {
	ticket := &Ticket{}
	if err := p.db.First(ticket, ticketID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return ticket, nil
}

func (p *PostgresStore) ListTickets() ([]Ticket, error) {
	var tickets []Ticket
	if err := p.db.Find(&tickets).Error; err != nil {
		return nil, err
	}

	if len(tickets) == 0 {
		tickets = []Ticket{}
	}

	return tickets, nil
}

func (p *PostgresStore) UpsertBoard(board *Board) error {
	result := p.db.Save(board)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (p *PostgresStore) GetBoard(boardID int) (*Board, error) {
	board := &Board{}
	if err := p.db.First(board, boardID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return board, nil
}

func (p *PostgresStore) ListBoards() ([]Board, error) {
	var boards []Board
	if err := p.db.Find(&boards).Error; err != nil {
		return nil, err
	}

	if len(boards) == 0 {
		boards = []Board{}
	}

	return boards, nil
}
