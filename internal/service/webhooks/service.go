package webhooks

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/psa"
	"github.com/thecoretg/ticketbot/internal/webex"
)

type Service struct {
	CWClient        *psa.Client
	WebexClient     *webex.Client
	WebexHookSecret string
	RootURL         string
}

func New(cw *psa.Client, wx *webex.Client, wxSecret, rootURL string) *Service {
	return &Service{
		CWClient:        cw,
		WebexClient:     wx,
		WebexHookSecret: wxSecret,
		RootURL:         rootURL,
	}
}

func (s *Service) ProcessAllHooks() error {
	start := time.Now()
	errored := false
	defer func() {
		if errored {
			slog.Error("hook sync complete with errors, see logs", "took_seconds", time.Since(start).Seconds())
		} else {
			slog.Info("hook sync complete", "took_seconds", time.Since(start).Seconds())
		}
	}()

	errch := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Go(func() {
		if err := s.ProcessCWHooks(); err != nil {
			errch <- fmt.Errorf("processing connectwise hooks: %w", err)
			return
		}
	})

	wg.Go(func() {
		if err := s.ProcessWebexHooks(); err != nil {
			errch <- fmt.Errorf("processing webex hooks: %w", err)
			return
		}
	})

	wg.Wait()
	close(errch)

	for err := range errch {
		if err != nil {
			errored = true
			slog.Error("hook sync", "error", err.Error())
		}
	}

	return nil
}

func (s *Service) ProcessCWHooks() error {
	p := map[string]string{
		"pageSize": "1000",
	}

	cwh, err := s.CWClient.ListCallbacks(p)
	if err != nil {
		return fmt.Errorf("listing connectwise callbacks: %w", err)
	}
	slog.Debug("hook sync: got existing connectwise callbacks", "total", len(cwh))

	if err := s.processCWHook(ticketsWebhookURL(s.RootURL), "ticket", "owner", 1, cwh); err != nil {
		return fmt.Errorf("processing ticketbot hook: %w", err)
	}

	return nil
}

func (s *Service) ProcessWebexHooks() error {
	errored := false
	hs, err := s.WebexClient.GetWebhooks(nil)
	if err != nil {
		return fmt.Errorf("listing webex webhooks: %w", err)
	}
	slog.Debug("webex hook sync: got existing webex hooks", "total", len(hs))

	aURL := fmt.Sprintf("%s/hooks/webex/attachmentActions", s.RootURL)
	errch := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := s.processWebexHook("TicketBot: Received Attachment Actions", aURL, "attachmentActions", "created", "", hs); err != nil {
			errch <- fmt.Errorf("processing webex attachment actions hook: %w", err)
		}
	})

	wg.Wait()
	close(errch)

	for err := range errch {
		if err != nil {
			errored = true
			slog.Error("webex hook sync", "error", err.Error())
		}
	}

	if errored {
		return errors.New("webex hook sync finished with errors, see logs")
	}

	return nil
}

func (s *Service) processCWHook(url, entity, level string, objectID int, currentHooks []psa.Callback) error {
	expected := psa.Callback{
		URL:      fmt.Sprintf("https://%s", url),
		Type:     entity,
		Level:    level,
		ObjectId: objectID,
	}

	found := false
	for _, h := range currentHooks {
		if h.URL == expected.URL {
			if cwHooksMatch(expected, h) && !found {
				slog.Debug("found existing callback", "id", h.ID, "entity", entity, "level", level, "url", url)
				found = true
				continue
			} else {
				if err := s.CWClient.DeleteCallback(h.ID); err != nil {
					return fmt.Errorf("deleting callback: %w", err)
				}
				slog.Info("hook sync: deleted unused callback", "id", h.ID, "url", h.URL)
			}
		}
	}

	if !found {
		newHook, err := s.CWClient.PostCallback(&expected)
		if err != nil {
			return fmt.Errorf("posting callback: %w", err)
		}
		slog.Info("hook sync: added new connectwise hook", "id", newHook.ID, "url", url, "entity", entity, "level", level, "objectID", objectID)
	}
	return nil
}

func (s *Service) processWebexHook(name, url, resource, event, filter string, currentHooks []webex.Webhook) error {
	expected := webex.Webhook{
		Name:      name,
		TargetURL: fmt.Sprintf("https://%s", url),
		Resource:  resource,
		Event:     event,
		Filter:    filter,
		Secret:    s.WebexHookSecret,
	}

	found := false
	for _, h := range currentHooks {
		if h.TargetURL == expected.TargetURL {
			if wxHooksMatch(expected, h) && !found {
				slog.Debug("found existing webex webhook", "id", h.ID, "resource", h.Resource, "event", h.Event, "filter", h.Filter, "url", h.TargetURL)
				found = true
				continue
			} else {
				if err := s.WebexClient.DeleteWebhook(h.ID); err != nil {
					return fmt.Errorf("deleting webex hook: %w", err)
				}
				slog.Info("webex webhook deleted", "id", h.ID, "url", h.TargetURL)
			}
		}
	}

	if !found {
		nh, err := s.WebexClient.CreateWebhook(&expected)
		if err != nil {
			return fmt.Errorf("posting webex webhook: %w", err)
		}
		slog.Info("added new webex hook", "id", nh.ID, "url", nh.TargetURL, "resource", nh.Resource, "event", nh.Event, "filter", nh.Filter)
	}

	return nil
}

func cwHooksMatch(expected, existing psa.Callback) bool {
	return expected.Type == existing.Type && expected.Level == existing.Level && expected.InactiveFlag == existing.InactiveFlag
}

func wxHooksMatch(expected, existing webex.Webhook) bool {
	if expected.Resource != existing.Resource {
		return false
	}

	if expected.Event != existing.Event {
		return false
	}

	if expected.Filter != existing.Filter {
		return false
	}

	if expected.Secret != existing.Secret {
		return false
	}

	return true
}

func ticketsWebhookURL(rootURL string) string {
	return fmt.Sprintf("%s/hooks/cw/tickets", rootURL)
}
