package webhooks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

type Service struct {
	CWClient *psa.Client
	RootURL  string
}

func New(cw *psa.Client, rootURL string) *Service {
	return &Service{
		CWClient: cw,
		RootURL:  rootURL,
	}
}

func (s *Service) ProcessAllHooks(ctx context.Context) error {
	start := time.Now()

	if err := s.ProcessCWHooks(ctx); err != nil {
		return fmt.Errorf("processing connectwise hooks: %w", err)
	}
	slog.Info("hook sync complete", "took_seconds", time.Since(start).Seconds())

	return nil
}

func (s *Service) ProcessCWHooks(ctx context.Context) error {
	_, err := s.EnsureTicketCallback(ctx)
	return err
}

// EnsureTicketCallback makes sure ConnectWise has exactly one ticket callback pointing at this
// instance, and reports whether it had to add it: true means ConnectWise had lost it, so no
// ticket webhooks were arriving.
func (s *Service) EnsureTicketCallback(ctx context.Context) (added bool, err error) {
	p := map[string]string{
		"pageSize": "1000",
	}

	cwh, err := s.CWClient.ListCallbacks(ctx, p)
	if err != nil {
		return false, fmt.Errorf("listing connectwise callbacks: %w", err)
	}
	slog.Debug("hook sync: got existing connectwise callbacks", "total", len(cwh))

	added, err = s.processCWHook(ctx, ticketsWebhookURL(s.RootURL), "ticket", "owner", 1, cwh)
	if err != nil {
		return false, fmt.Errorf("processing ticketbot hook: %w", err)
	}
	return added, nil
}

func (s *Service) processCWHook(ctx context.Context, url, entity, level string, objectID int, currentHooks []psa.Callback) (bool, error) {
	expected := psa.Callback{
		URL:      url,
		Type:     entity,
		Level:    level,
		ObjectID: objectID,
	}

	found := false
	for _, h := range currentHooks {
		if h.URL == expected.URL {
			if cwHooksMatch(expected, h) && !found {
				slog.Debug("found existing callback", "id", h.ID, "entity", entity, "level", level, "url", url)
				found = true
				continue
			} else {
				if err := s.CWClient.DeleteCallback(ctx, h.ID); err != nil {
					return false, fmt.Errorf("deleting callback: %w", err)
				}
				slog.Info("hook sync: deleted unused callback", "id", h.ID, "url", h.URL)
			}
		}
	}

	if !found {
		newHook, err := s.CWClient.PostCallback(ctx, &expected)
		if err != nil {
			return false, fmt.Errorf("posting callback: %w", err)
		}
		slog.Info("hook sync: added new connectwise hook", "id", newHook.ID, "url", url, "entity", entity, "level", level, "objectID", objectID)
		return true, nil
	}
	return false, nil
}

func cwHooksMatch(expected, existing psa.Callback) bool {
	return expected.Type == existing.Type && expected.Level == existing.Level && expected.InactiveFlag == existing.InactiveFlag
}

func ticketsWebhookURL(rootURL string) string {
	return fmt.Sprintf("%s/hooks/cw/tickets", rootURL)
}
