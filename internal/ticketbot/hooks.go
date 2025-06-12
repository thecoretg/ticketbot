package ticketbot

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"tctg-automation/pkg/connectwise"
)

func (s *server) initiateWebhook(ctx context.Context) error {
	currentHooks, err := s.cwClient.ListCallbacks(ctx, nil)
	if err != nil {
		return fmt.Errorf("listing callbacks: %w", err)
	}

	hook := &connectwise.Callback{
		URL:      s.ticketsWebhookURL(),
		Type:     "ticket",
		Level:    "owner",
		ObjectId: 1,
	}
	log.Printf("payload for webhook: %+v", hook)

	found := false
	for _, c := range currentHooks {
		if c.URL == hook.URL {
			if c.Type == hook.Type && c.Level == hook.Level && c.InactiveFlag == hook.InactiveFlag && !found {
				log.Printf("found existing webhook %d with URL %s", c.ID, c.URL)
				found = true
				continue
			} else {
				if err := s.cwClient.DeleteCallback(ctx, c.ID); err != nil {
					return fmt.Errorf("deleting webhook %d: %w", c.ID, err)
				}
				slog.Info("deleted unused webhook", "id", c.ID, "url", c.URL)
			}
		}
	}

	if !found {
		if _, err = s.cwClient.PostCallback(ctx, hook); err != nil {
			return fmt.Errorf("posting webhook: %w", err)
		}
	}

	return nil
}

func (s *server) ticketsWebhookURL() string {
	return fmt.Sprintf("%s/tickets", s.rootUrl)
}
