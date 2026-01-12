package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

func (s *Service) SyncWebexRecipients(ctx context.Context, maxSyncs int) error {
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
		_ = tx.Rollback(ctx)
	}()

	if err := txSvc.syncWebexRooms(ctx); err != nil {
		return fmt.Errorf("syncing webex rooms: %w", err)
	}

	if err := txSvc.syncWebexPeople(ctx, maxSyncs); err != nil {
		return fmt.Errorf("syncing webex people: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}

	return nil
}

func (s *Service) syncWebexRooms(ctx context.Context) error {
	start := time.Now()
	defer func() {
		slog.Info("webex room sync complete", "took_time", time.Since(start).Seconds())
	}()
	// get rooms from webex as source of truth
	wr, err := s.Webex.WebexClient.ListRooms(nil)
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

	for _, r := range roomsToRecipients(wr) {
		if _, err := s.Webex.Recipients.Upsert(ctx, r); err != nil {
			return fmt.Errorf("upserting room with name %s: %w", r.Name, err)
		}
	}

	return nil
}

func (s *Service) syncWebexPeople(ctx context.Context, maxSyncs int) error {
	start := time.Now()
	defer func() {
		slog.Info("webex people sync complete", "took_time", time.Since(start).Seconds())
	}()

	cwm, err := s.CW.CWClient.ListMembers(nil)
	if err != nil {
		return fmt.Errorf("getting members from connectwise: %w", err)
	}
	slog.Info("webex people sync: got members from connectwise", "total_members", len(cwm))

	sp, err := s.Webex.Recipients.ListPeople(ctx)
	if err != nil {
		return fmt.Errorf("listing people from store: %w", err)
	}
	slog.Info("webex people sync: got people from store", "total_people", len(sp))

	wp, err := s.getWxPeopleFromCwMembers(cwm, maxSyncs)
	if err != nil {
		return fmt.Errorf("getting webex people from connectwise members: %w", err)
	}

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

func (s *Service) getWxPeopleFromCwMembers(members []psa.Member, maxSyncs int) ([]webex.Person, error) {
	sem := make(chan struct{}, maxSyncs)
	var wg sync.WaitGroup
	errCh := make(chan error, len(members))

	var (
		wp []webex.Person
		mu sync.Mutex
	)

	for _, m := range members {
		sem <- struct{}{}
		wg.Add(1)
		go func(member psa.Member) {
			defer func() { <-sem }()
			defer wg.Done()

			if m.PrimaryEmail == "" {
				return
			}

			ppl, err := s.Webex.WebexClient.ListPeople(member.PrimaryEmail)
			if err != nil {
				errCh <- fmt.Errorf("listing people for email %s: %w", member.PrimaryEmail, err)
				return
			}

			if len(ppl) == 0 {
				return
			}

			mu.Lock()
			wp = append(wp, ppl[0])
			mu.Unlock()
		}(m)
	}

	wg.Wait()
	close(errCh)

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
