package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

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

func (h *Handler) GetStatus(statusID int) (*Status, error) {
	s := &Status{}
	if err := h.DB.Get(s, "SELECT * FROM status WHERE status_id = $1", statusID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting status by id: %w", err)
	}
	return s, nil
}

func (h *Handler) ListStatuses() ([]Status, error) {
	var statuses []Status
	if err := h.DB.Select(&statuses, "SELECT * FROM status"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing statuses: %w", err)
	}
	return statuses, nil
}

func (h *Handler) UpsertStatus(s *Status) error {
	_, err := h.DB.NamedExec(UpsertStatusSQL(), s)
	if err != nil {
		return fmt.Errorf("inserting status: %w", err)
	}
	slog.Info("status added or updated", "status_id", s.ID, "status_name", s.Name)
	return nil
}

func (h *Handler) DeleteStatus(statusID int) error {
	_, err := h.DB.Exec("DELETE FROM status WHERE status_id = $1", statusID)
	if err != nil {
		return err
	}
	slog.Info("status deleted", "status_id", statusID)
	return nil
}

func UpsertStatusSQL() string {
	return `INSERT INTO status (status_id, status_name, board_id, closed, active)
		VALUES (:status_id, :status_name, :board_id, :closed, :active)
		ON CONFLICT (status_id) DO UPDATE SET
			status_name = EXCLUDED.status_name,
			board_id = EXCLUDED.board_id,
			closed = EXCLUDED.closed,
			active = EXCLUDED.active`
}
