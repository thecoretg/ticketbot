package ticketbot

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type boardSetting struct {
	ID          int    `db:"id" json:"-"`
	BoardID     int    `db:"board_id" json:"board_id"`
	BoardName   string `db:"board_name" json:"board_name"`
	WebexRoomID string `db:"webex_room_id" json:"webex_room_id"`
	Enabled     bool   `db:"enabled" json:"enabled"`
}

type user struct {
	ID           int    `db:"id" json:"-"`
	CWId         string `db:"cw_id" json:"cw_id"`
	Email        string `db:"email" json:"email"`
	Mute         bool   `db:"mute" json:"mute"`
	IgnoreUpdate bool   `db:"ignore_update" json:"ignore_update"`
}

const (
	boardSettingsTableName = "boards"
	usersTableName         = "users"
)

var (
	boardSettingsSchema = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			board_id INTEGER NOT NULL,
			board_name TEXT NOT NULL,
			webex_room_id TEXT NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT 1,
			UNIQUE (board_id)
		);`, boardSettingsTableName)

	usersTableSchema = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			cw_id TEXT NOT NULL,
			email TEXT NOT NULL,
			mute BOOLEAN NOT NULL DEFAULT 0,
			ignore_update BOOLEAN NOT NULL DEFAULT 0,
			UNIQUE (cw_id, email)
		);`, usersTableName)
)

func initDB() (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite", "bot.db")
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	db.MustExec(boardSettingsSchema)
	db.MustExec(usersTableSchema)

	return db, nil
}

func getAllBoards(db *sqlx.DB) ([]boardSetting, error) {
	var boards []boardSetting
	err := db.Select(&boards, "SELECT * FROM boards")
	return boards, err
}

func getBoardByID(db *sqlx.DB, boardID int) (*boardSetting, error) {
	var board boardSetting
	err := db.Get(&board, "SELECT * FROM boards WHERE board_id = ?", boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting board by id: %w", err)
	}
	return &board, nil
}

func addOrUpdateBoard(db *sqlx.DB, board *boardSetting) (*boardSetting, error) {
	_, err := db.NamedExec(`
			INSERT INTO boards (board_id, board_name, webex_room_id, enabled)
		   	VALUES(:board_id, :board_name, :webex_room_id, :enabled)
		   	ON CONFLICT(board_id) DO UPDATE SET
				board_name = excluded.board_name,
				webex_room_id = excluded.webex_room_id,
				enabled = excluded.enabled
			`, board,
	)
	if err != nil {
		return nil, fmt.Errorf("adding or updating board: %w", err)
	}

	updatedBoard, err := getBoardByID(db, board.BoardID)
	if err != nil {
		return nil, fmt.Errorf("getting updated board by id: %w", err)
	}

	return updatedBoard, nil
}

func deleteBoard(db *sqlx.DB, boardID int) error {
	_, err := db.Exec("DELETE FROM boards WHERE board_id = ?", boardID)
	return err
}

func getAllUsers(db *sqlx.DB) ([]user, error) {
	var users []user
	err := db.Select(&users, "SELECT * FROM users")
	return users, err
}

func getUserByCwID(db *sqlx.DB, cwId string) (*user, error) {
	var user user
	err := db.Get(&user, "SELECT * FROM users WHERE cw_id = ?", cwId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No user found with the given cw_id
		}
		return nil, fmt.Errorf("getting user by cw_id: %w", err)
	}
	return &user, nil
}

func addOrUpdateUser(db *sqlx.DB, user *user) (*user, error) {
	_, err := db.NamedExec(`
			INSERT INTO users (cw_id, email, mute, ignore_update)
			VALUES(:cw_id, :email, :mute, :ignore_update)
			ON CONFLICT(cw_id, email) DO UPDATE SET
			    cw_id = excluded.cw_id,
			    email = excluded.email,
			    mute = excluded.mute,
			    ignore_update = excluded.ignore_update
			`, user,
	)
	if err != nil {
		return nil, fmt.Errorf("adding or updating user: %w", err)
	}

	updatedUser, err := getUserByCwID(db, user.CWId)
	if err != nil {
		return nil, fmt.Errorf("getting updated user by cw_id: %w", err)
	}

	return updatedUser, nil
}

func deleteUser(db *sqlx.DB, userCwId string) error {
	_, err := db.Exec("DELETE FROM users WHERE cw_id = ?", userCwId)
	return err
}
