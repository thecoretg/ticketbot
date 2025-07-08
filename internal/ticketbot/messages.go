package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"tctg-automation/internal/ticketbot/db"
	"tctg-automation/pkg/webex"
)

func (s *server) processMessageSent(c *gin.Context) {
	w := &webex.MessageWebhookBody{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("invalid request body: %w", err))
		return
	}

	msgID := w.Data.Id
	m, err := s.webexClient.GetMessage(c.Request.Context(), msgID)
	if err != nil {
		c.Error(fmt.Errorf("getting message details: %w", err))
		return
	}

	email := m.PersonEmail
	if email == "" {
		c.Error(errors.New("sender email is blank"))
		return
	}

	if email == s.webexBotEmail {
		// ignore messages sent by the bot itself, otherwise you get an infinite loop
		c.Status(http.StatusNoContent)
		return
	}

	sendMsg := false
	p := webex.MessagePostBody{
		Person: email,
	}

	cmd, subcmd, _ := parseMessageInput(m.Text)

	switch cmd {
	case "boards":
		switch subcmd {
		case "list":
			p.Markdown = s.makeBoardsListMsg()
			sendMsg = true
		}
	}

	if sendMsg {
		if err := s.webexClient.SendMessage(c.Request.Context(), p); err != nil {
			c.Error(fmt.Errorf("sending message: %w", err))
			return
		}
	}

	c.Status(http.StatusNoContent)
	return
}

func (s *server) processWebexRoomUpdate(ctx context.Context, roomID string) error {
	wr, err := s.webexClient.GetRoom(ctx, roomID)
	if err != nil {
		return fmt.Errorf("getting room from webex: %w", err)
	}

	r := db.NewWebexRoom(wr.Id, wr.Title)
	if err := s.dbHandler.UpsertWebexRoom(r); err != nil {
		return fmt.Errorf("inserting or updating webex room in db: %w", err)
	}

	return nil
}

func (s *server) ensureWebexRoomExists(roomID string, name string) error {
	c, err := s.dbHandler.GetWebexRoom(roomID)
	if err != nil {
		return fmt.Errorf("querying db for webex room: %w", err)
	}

	if c == nil {
		r := db.NewWebexRoom(roomID, name)
		if err := s.dbHandler.UpsertWebexRoom(r); err != nil {
			return fmt.Errorf("inserting new webex room into db: %w", err)
		}
	}

	return nil
}

func (s *server) makeBoardsListMsg() string {
	boards, err := s.dbHandler.ListBoards()
	if err != nil {
		return "An error occured getting the boards. Please notify an admin."
	}

	if len(boards) == 0 {
		return "No boards were found."
	}

	var lines []string
	for _, board := range boards {
		l := fmt.Sprintf("**%s** (id: %d, notifications: %v)", board.Name, board.ID, board.NotifyEnabled)
		lines = append(lines, l)
	}

	return strings.Join(lines, "\n")
}

func parseMessageInput(msg string) (command string, subcommand string, args map[string]string) {
	parts := strings.Fields(msg)

	if len(parts) > 0 {
		command = parts[0]
	}

	if len(parts) > 1 && !strings.HasPrefix(parts[1], "--") {
		subcommand = parts[1]
	}

	start := 1
	if subcommand != "" {
		start = 2
	}

	for i := start; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "--") && i+1 < len(parts) {
			key := strings.TrimPrefix(parts[i], "--")
			value := parts[i+1]
			args[key] = value
			i++
		}
	}

	return
}
