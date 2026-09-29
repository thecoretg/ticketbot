package ticketbot

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
)

type purgeTickets struct {
	repos.TicketRepository
	calls []string
}

func (f *purgeTickets) DeleteSoftDeletedOlderThan(_ context.Context, days int) (int64, error) {
	f.calls = append(f.calls, "soft-deleted:"+strconv.Itoa(days))
	return 0, nil
}

func (f *purgeTickets) DeleteClosedOlderThan(_ context.Context, days int) (int64, error) {
	f.calls = append(f.calls, "closed:"+strconv.Itoa(days))
	return 0, nil
}

type purgeCfg struct{ history, closed int }

func (c purgeCfg) GetHistoryRetentionDays() int { return c.history }
func (c purgeCfg) ClosedTicketRetention() int   { return c.closed }

func TestTicketPurger(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  purgeCfg
		want []string
	}{
		{"closed retention off", purgeCfg{history: 90}, []string{"soft-deleted:90"}},
		{"closed retention on", purgeCfg{history: 90, closed: 60}, []string{"soft-deleted:90", "closed:60"}},
		{"history kept forever keeps soft-deleted tickets", purgeCfg{closed: 60}, []string{"closed:60"}},
		{"nothing to do", purgeCfg{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &purgeTickets{}
			(&TicketPurger{Tickets: repo, Cfg: tc.cfg}).Purge(context.Background(), time.Now())
			if !reflect.DeepEqual(repo.calls, tc.want) {
				t.Errorf("calls = %v, want %v", repo.calls, tc.want)
			}
		})
	}
}
