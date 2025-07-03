package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

type WebexSpace struct {
	ID   string `db:"space_id"`
	Name string `db:"space_name"`
}

func NewWebexSpace(id, name string) *WebexSpace {
	return &WebexSpace{
		ID:   id,
		Name: name,
	}
}

func (h *Handler) GetWebexSpace(spaceID string) (*WebexSpace, error) {
	s := &WebexSpace{}
	if err := h.DB.Get(s, "SELECT * FROM webex_space WHERE space_id = $1", spaceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting webex space by id: %w", err)
	}

	return s, nil
}

func (h *Handler) ListWebexSpaces() ([]WebexSpace, error) {
	var spaces []WebexSpace
	if err := h.DB.Select(&spaces, "SELECT * FROM webex_space"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing webex spaces: %w", err)
	}
	return spaces, nil
}

func (h *Handler) UpsertWebexSpace(s *WebexSpace) error {
	_, err := h.DB.NamedExec(UpsertWebexSpaceSQL(), s)
	if err != nil {
		return fmt.Errorf("inserting webex space: %w", err)
	}
	slog.Info("webex space added or updated", "space_id", s.ID, "space_name", s.Name)
	return nil
}

func (h *Handler) DeleteWebexSpace(spaceID string) error {
	_, err := h.DB.Exec("DELETE FROM board WHERE board_id = $1", spaceID)
	if err != nil {
		return err
	}
	slog.Info("webex space deleted", "space_id", spaceID)
	return nil
}

func UpsertWebexSpaceSQL() string {
	return `INSERT INTO webex_space (space_id, space_name)
		VALUES (:space_id, :space_name)
		ON CONFLICT (:space_id) DO UPDATE SET
			space_name = EXCLUDED.space_name`
}
