// Package journal records a human-readable, per-ticket lifecycle audit trail. Each
// ticket has one journal record; every non-bot processing run appends a friendly
// timeline entry. It is the default place to answer "what happened to this ticket?",
// leaving slog for granular debugging.
package journal

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// maxRunsPerTicket caps the timeline so a frequently-updated ticket can't grow
// unbounded; the oldest runs drop off.
const maxRunsPerTicket = 50

type Service struct {
	repo repos.TicketJournalRepository
	cfg  *models.Config
}

// overviewLimit caps how many ticket rows the overview table loads.
const overviewLimit = 500

func New(repo repos.TicketJournalRepository, cfg *models.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// ListSummaries returns recent ticket journals (without their run timelines) for
// the overview table, newest activity first.
func (s *Service) ListSummaries(ctx context.Context) ([]*models.TicketJournal, error) {
	return s.repo.ListSummaries(ctx, overviewLimit)
}

// Get returns a single ticket's full journal including its run timeline.
func (s *Service) Get(ctx context.Context, ticketID int) (*models.TicketJournal, error) {
	return s.repo.Get(ctx, ticketID)
}

// Record upserts the ticket's journal: it refreshes the denormalized snapshot
// columns from the post-sync FullTicket (nil-safe — a nil full keeps whatever
// snapshot already exists, e.g. when a sync failed), appends the run, and caps the
// timeline to maxRunsPerTicket.
func (s *Service) Record(ctx context.Context, ticketID int, full *models.FullTicket, run models.TicketRun) error {
	j, err := s.repo.Get(ctx, ticketID)
	if err != nil {
		j = nil // ErrTicketJournalNotFound or a read error — start fresh; the upsert is authoritative
	}
	if j == nil {
		j = &models.TicketJournal{TicketID: ticketID, FirstSeen: run.Time}
	}

	if full != nil {
		j.Summary = full.Ticket.Summary
		j.BoardName = full.Board.Name
		j.CompanyName = full.Company.Name
		j.ContactName = contactName(full.Contact)
		j.StatusName = full.Status.Name
		j.OwnerName = memberName(full.Owner)
		j.TypeName = typeName(full.Type)
		j.SubtypeName = subTypeName(full.SubType)
		j.ItemName = itemName(full.Item)
	}

	j.LastTrigger = run.Trigger
	j.LastOutcome = run.Outcome
	j.HadError = run.HadError
	j.LastRun = run.Time

	j.Runs = append(j.Runs, run)
	if len(j.Runs) > maxRunsPerTicket {
		j.Runs = j.Runs[len(j.Runs)-maxRunsPerTicket:]
	}

	if _, err := s.repo.Upsert(ctx, j); err != nil {
		return err
	}
	return nil
}

// StartCleanup launches a goroutine that periodically deletes journals untouched
// for longer than the configured log-retention period. Mirrors the log persister's
// cleanup cadence.
func (s *Service) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		clean := func() {
			days := s.cfg.LogRetentionDays
			if days <= 0 {
				return
			}
			n, err := s.repo.DeleteOlderThan(ctx, time.Now().AddDate(0, 0, -days))
			if err != nil {
				slog.Warn("journal cleanup failed", "error", err.Error())
				return
			}
			if n > 0 {
				slog.Info("journal cleanup removed old ticket journals", "deleted", n, "retention_days", days)
			}
		}

		clean()
		for {
			select {
			case <-ticker.C:
				clean()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func contactName(c *models.Contact) string {
	if c == nil {
		return ""
	}
	name := c.FirstName
	if c.LastName != nil && *c.LastName != "" {
		name = strings.TrimSpace(name + " " + *c.LastName)
	}
	return name
}

func memberName(m *models.Member) string {
	if m == nil {
		return ""
	}
	name := strings.TrimSpace(m.FirstName + " " + m.LastName)
	if name == "" {
		return m.Identifier
	}
	return name
}

func typeName(t *models.TicketType) string {
	if t == nil {
		return ""
	}
	return t.Name
}

func subTypeName(t *models.TicketSubType) string {
	if t == nil {
		return ""
	}
	return t.Name
}

func itemName(t *models.TicketItem) string {
	if t == nil {
		return ""
	}
	return t.Name
}
