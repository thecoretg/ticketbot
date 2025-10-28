package server

import (
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/psa"
)

func (cl *Client) initAllHooks() error {
	// this will eventually include webex hooks
	return cl.initiateCWHooks()
}

func (cl *Client) initiateCWHooks() error {
	params := map[string]string{
		"pageSize": "1000",
	}
	cwHooks, err := cl.CWClient.ListCallbacks(params)
	if err != nil {
		return fmt.Errorf("listing callbacks: %w", err)
	}
	slog.Debug("got existing webhooks", "total", len(cwHooks))

	if err := cl.processCwHook(cl.ticketsWebhookURL(), "ticket", "owner", 1, cwHooks); err != nil {
		return fmt.Errorf("processing tickets hook: %w", err)
	}

	return nil
}

func (cl *Client) processCwHook(url, entity, level string, objectID int, currentHooks []psa.Callback) error {
	hook := &psa.Callback{
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
				if err := cl.CWClient.DeleteCallback(c.ID); err != nil {
					return fmt.Errorf("deleting webhook %d: %w", c.ID, err)
				}
				slog.Debug("deleted unused connectwise webhook", "id", c.ID, "url", c.URL)
			}
		}
	}

	if !found {
		newHook, err := cl.CWClient.PostCallback(hook)
		if err != nil {
			return fmt.Errorf("posting webhook: %w", err)
		}
		slog.Debug("added new connectwise hook", "id", newHook.ID, "url", url, "entity", entity, "level", level, "objectID", objectID)
	}
	return nil
}

func (cl *Client) ticketsWebhookURL() string {
	return fmt.Sprintf("%s/hooks/cw/tickets", cl.Config.RootURL)
}
