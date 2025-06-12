package ticketbot

import (
	"log"
	"testing"
)

func TestDB(t *testing.T) {
	db, err := initDB()
	if err != nil {
		log.Fatalf("initializing db: %v", err)
	}

	b := &boardSetting{
		BoardID:     1,
		BoardName:   "Test",
		WebexRoomID: "abcdefg",
		Enabled:     true,
	}
	if err := addOrUpdateBoard(db, b); err != nil {
		log.Fatalf("adding board setting: %v", err)
	}

	boards, err := getAllBoards(db)
	if err != nil {
		log.Fatalf("getting all boards: %v", err)
	}

	for _, board := range boards {
		log.Printf("Got board %d with name %s", board.BoardID, board.BoardName)
	}

	if err := deleteBoard(db, b.BoardID); err != nil {
		log.Fatalf("deleting board: %v", err)
	}

	log.Printf("Deleted board with ID %d", b.BoardID)

	boards, err = getAllBoards(db)
	if err != nil {
		log.Fatalf("getting all boards after deletion: %v", err)
	}

	for _, board := range boards {
		log.Printf("Remaining board %d with name %s", board.BoardID, board.BoardName)
	}

	u := &user{
		ID:    1,
		CWId:  "SomeGuy",
		Email: "someguy@somedomain.com",
		Mute:  false,
	}

	if err := addOrUpdateUser(db, u); err != nil {
		log.Fatalf("adding user: %v", err)
	}

	users, err := getAllUsers(db)
	if err != nil {
		log.Fatalf("getting all users: %v", err)
	}

	for _, user := range users {
		log.Printf("Got user with cw id %s and email %s", user.CWId, user.Email)
	}

	nonExistUser, err := getUserByCwID(db, "NonExistentUser")
	if err != nil {
		log.Printf("expected no error when getting non-existent user: %v", err)
	}

	if nonExistUser == nil {
		log.Println("Successfully returned nil for non-existent user")
	} else {
		log.Fatalf("Expected nil for non-existent user, got: %+v", nonExistUser)
	}

	if err := deleteUser(db, u.CWId); err != nil {
		log.Fatalf("deleting user: %v", err)
	}

	log.Printf("Deleted user with CW ID %s", u.CWId)
}
