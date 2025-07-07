package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

type Board struct {
	ID            int     `db:"board_id" json:"id"`
	Name          string  `db:"board_name" json:"name"`
	NotifyEnabled bool    `db:"notify_enabled" json:"notify_enabled"`
	WebexSpace    *string `db:"webex_space_id" json:"webex_space"`
}

// NewBoard creates a board, but does not handle notify settings
func NewBoard(id int, name string) *Board {
	return &Board{
		ID:   id,
		Name: name,
	}
}

func (h *Handler) SetNotifySpace(b *Board, spaceID *string, enabled bool) {
	b.WebexSpace = spaceID
	b.NotifyEnabled = enabled
	slog.Info("notify space set for board", "board_id", b.ID, "board_name", b.Name, "space_id", spaceID, "enabled", enabled)
}

func (h *Handler) GetBoard(boardID int) (*Board, error) {
	b := &Board{}
	if err := h.DB.Get(b, "SELECT * FROM board WHERE board_id = $1", boardID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting board by id: %w", err)
	}

	return b, nil
}

func (h *Handler) ListBoards() ([]Board, error) {
	var boards []Board
	if err := h.DB.Select(&boards, "SELECT * FROM board"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing boards: %w", err)
	}
	return boards, nil
}

func (h *Handler) UpsertBoard(b *Board) error {
	_, err := h.DB.NamedExec(UpsertBoardSQL(), b)
	if err != nil {
		return fmt.Errorf("inserting board: %w", err)
	}
	slog.Info("board added or updated", "board_id", b.ID, "board_name", b.Name)
	return nil
}

func (h *Handler) DeleteBoard(boardID int) error {
	_, err := h.DB.Exec("DELETE FROM board WHERE board_id = $1", boardID)
	if err != nil {
		return fmt.Errorf("deleting board: %w", err)
	}
	slog.Info("board deleted", "board_id", boardID, "board_name", boardID)
	return nil
}

func UpsertBoardSQL() string {
	return `INSERT INTO board (board_id, board_name)
		VALUES (:board_id, :board_name)
		ON CONFLICT (board_id) DO UPDATE SET
			board_name = EXCLUDED.board_name`
}

func (h *Handler) UpdateBoardNotifySettings(boardID int, notifyEnabled bool, spaceID *string) error {
	_, err := h.DB.Exec("UPDATE board SET notify_enabled = $1, webex_space_id = $2 WHERE board_id = $3", notifyEnabled, spaceID, boardID)
	if err != nil {
		return fmt.Errorf("updating notify settings: %w", err)
	}
	slog.Info("notify settings updated", "board_id", boardID, "notify_enabled", notifyEnabled, "space_id", spaceID)
	return nil
}
