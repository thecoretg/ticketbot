package ticketbot

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/connectwise"
)

func (s *Server) InitAllHooks() error {
	// this will eventually include webex hooks
	return s.initiateCWHooks()
}

func (s *Server) initiateCWHooks() error {
	params := map[string]string{
		"pageSize": "1000",
	}
	cwHooks, err := s.CWClient.ListCallbacks(params)
	if err != nil {
		return fmt.Errorf("listing callbacks: %w", err)
	}
	slog.Debug("got existing webhooks", "total", len(cwHooks))

	if err := s.processCwHook(s.ticketsWebhookURL(), "ticket", "owner", 1, cwHooks); err != nil {
		return fmt.Errorf("processing tickets hook: %w", err)
	}

	return nil
}

func (s *Server) processCwHook(url, entity, level string, objectID int, currentHooks []connectwise.Callback) error {
	hook := &connectwise.Callback{
		URL:      fmt.Sprintf("https://%s", url),
		Type:     entity,
		Level:    level,
		ObjectId: objectID,
	}

	slog.Debug("expected connectwise webhook", "url", url, "entity", entity, "level", level, "objectID", objectID)
	found := false
	for _, c := range currentHooks {
		slog.Debug("checking connectwise webhook", "id", c.ID, "url", c.URL, "type", c.Type, "level", c.Level, "inactiveFlag", c.InactiveFlag)
		if c.URL == hook.URL {
			slog.Debug("found matching url for webhook")
			if c.Type == hook.Type && c.Level == hook.Level && c.InactiveFlag == hook.InactiveFlag && !found {
				slog.Debug("found existing connectwise webhook", "id", c.ID, "entity", entity, "level", level, "url", url)
				found = true
				continue
			} else {
				if err := s.CWClient.DeleteCallback(c.ID); err != nil {
					return fmt.Errorf("deleting webhook %d: %w", c.ID, err)
				}
				slog.Info("deleted unused connectwise webhook", "id", c.ID, "url", c.URL)
			}
		}
	}

	if !found {
		newHook, err := s.CWClient.PostCallback(hook)
		if err != nil {
			return fmt.Errorf("posting webhook: %w", err)
		}
		slog.Info("added new connectwise hook", "id", newHook.ID, "url", url, "entity", entity, "level", level, "objectID", objectID)
	}
	return nil
}

func (s *Server) ticketsWebhookURL() string {
	return fmt.Sprintf("%s/hooks/cw/tickets", s.Config.RootURL)
}

func requireValidCWSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Error(fmt.Errorf("reading request body: %w", err))
			c.Next()
			c.Abort()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		valid, err := connectwise.ValidateWebhook(c.Request)
		if err != nil || !valid {
			c.Error(fmt.Errorf("invalid ConnectWise webhook signature: %w", err))
			c.Next()
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body for further processing
		c.Next()
	}
}
