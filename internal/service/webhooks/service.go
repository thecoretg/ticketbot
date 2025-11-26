package webhooks

import (
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/pkg/psa"
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
			slog.Debug("found matching url for webhook")
			if hooksMatch(expected, h) && !found {
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

func hooksMatch(expected, existing psa.Callback) bool {
	return expected.Type == existing.Type && expected.Level == existing.Level && expected.InactiveFlag == existing.InactiveFlag
}

func ticketsWebhookURL(rootURL string) string {
	return fmt.Sprintf("%s/hooks/cw/tickets", rootURL)
}
