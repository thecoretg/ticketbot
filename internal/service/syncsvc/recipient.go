package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/models"
)

// SyncWebexRecipients reports into ph. Its count is CW members looked up in Webex, the slow
// part; the room sync before it shows as a label only.
func (s *Service) SyncWebexRecipients(ctx context.Context, maxSyncs int, ph *phase) error {
	slog.Info("beginning webex room sync")
	start := time.Now()
	defer func() {
		slog.Info("full webex recipient sync complete", "took_time", time.Since(start).Seconds())
	}()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}

	txSvc := s.withTx(tx)

	defer func() {
		// not ctx: after a cancel the rollback still has to reach the database
		_ = tx.Rollback(context.WithoutCancel(ctx))
	}()

	if err := txSvc.syncWebexRooms(ctx, ph); err != nil {
		return fmt.Errorf("syncing webex rooms: %w", err)
	}

	if err := txSvc.syncWebexPeople(ctx, maxSyncs, ph); err != nil {
		return fmt.Errorf("syncing webex people: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}

	return nil
}

func (s *Service) syncWebexRooms(ctx context.Context, ph *phase) error {
	start := time.Now()
	defer func() {
		slog.Info("webex room sync complete", "took_time", time.Since(start).Seconds())
	}()
	// get rooms from webex as source of truth
	wr, err := s.Webex.WebexClient.ListRooms(ctx, nil)
	if err != nil {
		return fmt.Errorf("getting rooms from webex: %w", err)
	}
	slog.Info("webex room sync: got rooms from webex", "total_rooms", len(wr))

	// get current rooms from store
	sr, err := s.Webex.Recipients.ListRooms(ctx)
	if err != nil {
		return fmt.Errorf("getting rooms from store: %w", err)
	}
	slog.Info("webex room sync: got rooms from store", "total_rooms", len(sr))

	ph.label("Saving Webex rooms")

	for _, r := range roomsToRecipients(wr) {
		if _, err := s.Webex.Recipients.Upsert(ctx, r); err != nil {
			return fmt.Errorf("upserting room with name %s: %w", r.Name, err)
		}
	}

	return nil
}

func (s *Service) syncWebexPeople(ctx context.Context, maxSyncs int, ph *phase) error {
	ph.fetching("Fetching members from ConnectWise")
	start := time.Now()
	defer func() {
		slog.Info("webex people sync complete", "took_time", time.Since(start).Seconds())
	}()

	cwm, err := s.CW.CWClient.ListMembers(ctx, nil)
	if err != nil {
		return fmt.Errorf("getting members from connectwise: %w", err)
	}
	slog.Info("webex people sync: got members from connectwise", "total_members", len(cwm))

	sp, err := s.Webex.Recipients.ListPeople(ctx)
	if err != nil {
		return fmt.Errorf("listing people from store: %w", err)
	}
	slog.Info("webex people sync: got people from store", "total_people", len(sp))

	ph.counting("Looking up members in Webex", len(cwm))
	wp, err := s.getWxPeopleFromCwMembers(ctx, cwm, maxSyncs, ph)
	if err != nil {
		return fmt.Errorf("getting webex people from connectwise members: %w", err)
	}

	ph.label("Saving Webex people")

	for _, p := range peopleToRecipients(wp) {
		if _, err := s.Webex.Recipients.Upsert(ctx, p); err != nil {
			return fmt.Errorf("upserting person with name %s: %w", p.Name, err)
		}
	}

	for _, d := range peopleToDelete(cwm, sp) {
		if err := s.Webex.Recipients.Delete(ctx, d.ID); err != nil {
			return fmt.Errorf("deleting person with id %d (%s): %w", d.ID, d.Name, err)
		}
	}

	return nil
}

func (s *Service) getWxPeopleFromCwMembers(ctx context.Context, members []psa.Member, maxSyncs int, ph *phase) ([]webex.Person, error) {
	errCh := make(chan error, len(members))

	var (
		wp []webex.Person
		mu sync.Mutex
	)

	stopped := forEach(ctx, members, maxSyncs, func(member psa.Member) {
		if member.PrimaryEmail == "" {
			ph.step(nil)
			return
		}

		ppl, err := s.Webex.WebexClient.ListPeople(ctx, member.PrimaryEmail)
		if err != nil {
			err = fmt.Errorf("listing people for email %s: %w", member.PrimaryEmail, err)
			ph.step(err)
			errCh <- err
			return
		}
		ph.step(nil)

		if len(ppl) == 0 {
			return
		}

		mu.Lock()
		wp = append(wp, ppl[0])
		mu.Unlock()
	})
	close(errCh)

	if stopped != nil {
		return nil, stopped
	}

	if len(errCh) > 0 {
		return nil, <-errCh
	}

	return wp, nil
}

func peopleToRecipients(webexPpl []webex.Person) []*models.WebexRecipient {
	var toUpsert []*models.WebexRecipient
	for _, p := range webexPpl {
		if len(p.Emails) == 0 {
			continue
		}

		r := &models.WebexRecipient{
			WebexID:      p.ID,
			Name:         p.DisplayName,
			Email:        &p.Emails[0],
			Type:         "person",
			LastActivity: p.LastActivity,
		}

		toUpsert = append(toUpsert, r)
	}

	return toUpsert
}

func roomsToRecipients(webexRooms []webex.Room) []*models.WebexRecipient {
	var toUpsert []*models.WebexRecipient
	for _, w := range webexRooms {
		if w.Type != "group" {
			continue
		}

		r := &models.WebexRecipient{
			WebexID:      w.ID,
			Name:         w.Title,
			Type:         "room",
			LastActivity: w.LastActivity,
		}
		toUpsert = append(toUpsert, r)
	}

	return toUpsert
}

func peopleToDelete(cwMembers []psa.Member, storedPpl []*models.WebexRecipient) []*models.WebexRecipient {
	l := make(map[string]struct{})
	for _, m := range cwMembers {
		l[m.PrimaryEmail] = struct{}{}
	}

	var toDelete []*models.WebexRecipient
	for _, p := range storedPpl {
		if _, ok := l[*p.Email]; !ok {
			toDelete = append(toDelete, p)
		}
	}

	return toDelete
}
