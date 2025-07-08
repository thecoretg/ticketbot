package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

type WebexRoom struct {
	ID   string `db:"room_id"`
	Name string `db:"room_name"`
}

func NewWebexRoom(id, name string) *WebexRoom {
	return &WebexRoom{
		ID:   id,
		Name: name,
	}
}

func (h *Handler) GetWebexRoom(roomID string) (*WebexRoom, error) {
	s := &WebexRoom{}
	if err := h.DB.Get(s, "SELECT * FROM webex_room WHERE room_id = $1", roomID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting webex room by id: %w", err)
	}

	return s, nil
}

func (h *Handler) ListWebexRooms() ([]WebexRoom, error) {
	var rooms []WebexRoom
	if err := h.DB.Select(&rooms, "SELECT * FROM webex_room"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing webex rooms: %w", err)
	}
	return rooms, nil
}

func (h *Handler) UpsertWebexRoom(r *WebexRoom) error {
	_, err := h.DB.NamedExec(UpsertWebexRoomSQL(), r)
	if err != nil {
		return fmt.Errorf("inserting webex room: %w", err)
	}
	slog.Info("webex room added or updated", "room_id", r.ID, "room_nane", r.Name)
	return nil
}

func (h *Handler) DeleteWebexRoom(roomID string) error {
	_, err := h.DB.Exec("DELETE FROM board WHERE board_id = $1", roomID)
	if err != nil {
		return err
	}
	slog.Info("webex room deleted", "room_id", roomID)
	return nil
}

func UpsertWebexRoomSQL() string {
	return `INSERT INTO webex_room (room_id, room_name)
		VALUES (:room_id, :room_name)
		ON CONFLICT (:room_id) DO UPDATE SET
			room_name = EXCLUDED.room_name`
}
