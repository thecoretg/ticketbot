package ticketbot

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"
)

func (s *server) initiateAllHooks(ctx context.Context) error {
	if err := s.initiateCWHooks(); err != nil {
		return fmt.Errorf("initiating connectwise hooks: %w", err)
	}

	if err := s.initiateWebexHooks(ctx); err != nil {
		return fmt.Errorf("initiating webex hooks: %w", err)
	}

	return nil
}

func (s *server) initiateCWHooks() error {
	cwHooks, err := s.cwClient.ListCallbacks(nil)
	if err != nil {
		return fmt.Errorf("listing callbacks: %w", err)
	}

	if err := s.processCwHook(s.ticketsWebhookURL(), "ticket", "owner", 1, cwHooks); err != nil {
		return fmt.Errorf("processing tickets hook: %w", err)
	}

	return nil
}

func (s *server) initiateWebexHooks(ctx context.Context) error {
	wh, err := s.webexClient.GetAllWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("listing webhooks: %w", err)
	}

	filter := "roomType=direct"
	if err := s.processWebexHook(ctx, "Ticketbot: Message Created", s.messageWebhookURL(), "messages", "created", &filter, wh); err != nil {
		return fmt.Errorf("processing messages hook: %w", err)
	}

	return nil
}
func (s *server) processCwHook(url, entity, level string, objectID int, currentHooks []connectwise.Callback) error {
	hook := &connectwise.Callback{
		URL:      url,
		Type:     entity,
		Level:    level,
		ObjectId: objectID,
	}

	found := false
	for _, c := range currentHooks {
		if c.URL == hook.URL {
			if c.Type == hook.Type && c.Level == hook.Level && c.InactiveFlag == hook.InactiveFlag && !found {
				slog.Debug("found existing connectwise webhook", "id", c.ID, "entity", entity, "level", level, "url", url)
				found = true
				continue
			} else {
				if err := s.cwClient.DeleteCallback(c.ID); err != nil {
					return fmt.Errorf("deleting webhook %d: %w", c.ID, err)
				}
				slog.Debug("deleted unused connectwise webhook", "id", c.ID, "url", c.URL)
			}
		}
	}

	if !found {
		if _, err := s.cwClient.PostCallback(hook); err != nil {
			return fmt.Errorf("posting webhook: %w", err)
		}
		slog.Info("added new connectwise hook", "url", url, "entity", entity, "level", level, "objectID", objectID)
	}
	return nil
}

func (s *server) processWebexHook(ctx context.Context, name, url, resource, event string, filter *string, currentHooks []webex.Webhook) error {
	hook := &webex.Webhook{
		Name:      name,
		TargetUrl: url,
		Resource:  resource,
		Event:     event,
		Secret:    s.webexSecret,
	}
	if filter != nil {
		hook.Filter = *filter
	}

	found := false
	for _, c := range currentHooks {
		if c.TargetUrl == hook.TargetUrl {
			if c.Name == hook.Name && c.Resource == hook.Resource && c.Event == hook.Event && c.Filter == hook.Filter && !found {
				slog.Debug("found existing webex webhook", "name", c.Name, "url", c.TargetUrl, "resource", c.Resource, "event", c.Event, "filter", c.Event)
				found = true
				continue
			} else {
				if err := s.webexClient.DeleteWebhook(ctx, c.ID); err != nil {
					return fmt.Errorf("deleting webhook: %w", err)
				}
				slog.Debug("deleted unused webex webhook", "id", c.ID, "url", c.TargetUrl)
			}
		}
	}

	if !found {
		if _, err := s.webexClient.CreateWebhook(ctx, hook); err != nil {
			return fmt.Errorf("posting hook: %w", err)
		}
		slog.Info("added new hook", "url", url, "resource", resource, "event", event)
	}

	return nil
}

func (s *server) ticketsWebhookURL() string {
	return fmt.Sprintf("%s/hooks/cw/tickets", s.rootUrl)
}

func (s *server) contactsWebhookURL() string {
	return fmt.Sprintf("%s/hooks/cw/contacts", s.rootUrl)
}

func (s *server) companiesWebhookURL() string {
	return fmt.Sprintf("%s/hooks/cw/companies", s.rootUrl)
}

func (s *server) membersWebhookURL() string {
	return fmt.Sprintf("%s/hooks/cw/members", s.rootUrl)
}

func (s *server) messageWebhookURL() string {
	return fmt.Sprintf("%s/hooks/webex/messages", s.rootUrl)
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

func (s *server) requireValidWebexSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Error(fmt.Errorf("reading request body: %w", err))
			c.Next()
			c.Abort()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		valid, err := webex.ValidateWebhook(c.Request, s.webexSecret)
		if err != nil || !valid {
			c.Error(fmt.Errorf("invalid Webex webhook signature: %w", err))
			c.Next()
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		c.Next()
	}
}
