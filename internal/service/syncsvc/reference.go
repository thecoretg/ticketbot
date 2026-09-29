package syncsvc

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// idBatch is how many ids go into one "id in (...)" condition, which keeps the request URL
// well under ConnectWise's length limit.
const idBatch = 100

// SyncMembers upserts every ConnectWise member and soft-deletes stored members ConnectWise no
// longer has. Like the board sync it writes in one transaction, so a failure saves nothing.
func (s *Service) SyncMembers(ctx context.Context, ph *phase) error {
	cwm, err := s.CW.CWClient.ListMembers(ctx, map[string]string{
		"fields":   "id,identifier,firstName,lastName,primaryEmail",
		"pageSize": "1000",
	})
	if err != nil {
		return fmt.Errorf("listing connectwise members: %w", err)
	}
	stored, err := s.CW.Members.List(ctx)
	if err != nil {
		return fmt.Errorf("listing members from store: %w", err)
	}

	return s.inTx(ctx, func(tx *Service) error {
		ph.counting("Syncing members", len(cwm))
		seen := make(map[int]bool, len(cwm))
		for _, m := range cwm {
			if err := ctx.Err(); err != nil {
				return err
			}
			seen[m.ID] = true
			_, err := tx.CW.Members.Upsert(ctx, &models.Member{
				ID: m.ID, Identifier: m.Identifier, FirstName: m.FirstName, LastName: m.LastName, PrimaryEmail: m.PrimaryEmail,
			})
			if err != nil {
				err = fmt.Errorf("upserting member %d (%s): %w", m.ID, m.Identifier, err)
			}
			ph.step(err)
		}
		ph.label("Saving members")
		for _, m := range stored {
			if !m.Deleted && !seen[m.ID] {
				if err := tx.CW.Members.SoftDelete(ctx, m.ID); err != nil {
					return fmt.Errorf("soft deleting member %d (%s): %w", m.ID, m.Identifier, err)
				}
			}
		}
		return nil
	})
}

// SyncCompanies refreshes the companies ticketbot has stored: their names change, and a company
// deleted in ConnectWise is soft-deleted here. Companies nobody's ticket references are not
// fetched; they arrive with the first ticket that does.
func (s *Service) SyncCompanies(ctx context.Context, ph *phase) error {
	stored, err := s.CW.Companies.List(ctx)
	if err != nil {
		return fmt.Errorf("listing companies from store: %w", err)
	}
	ids := liveIDs(stored, func(c *models.Company) (int, bool) { return c.ID, c.Deleted })
	ph.fetching(fmt.Sprintf("Fetching %d stored companies from ConnectWise", len(ids)))

	found := make(map[int]psa.Company, len(ids))
	for batch := range slices.Chunk(ids, idBatch) {
		cwc, err := s.CW.CWClient.ListCompanies(ctx, map[string]string{
			"conditions": idCondition(batch),
			"fields":     "id,name,deletedFlag",
			"pageSize":   "1000",
		})
		if err != nil {
			return fmt.Errorf("listing companies from connectwise: %w", err)
		}
		for _, c := range cwc {
			found[c.ID] = c
		}
	}

	return s.inTx(ctx, func(tx *Service) error {
		ph.counting("Syncing companies", len(ids))
		for _, id := range ids {
			if err := ctx.Err(); err != nil {
				return err
			}
			c, ok := found[id]
			if !ok || c.DeletedFlag {
				if err := tx.CW.Companies.SoftDelete(ctx, id); err != nil {
					ph.step(fmt.Errorf("soft deleting company %d: %w", id, err))
					continue
				}
				ph.step(nil)
				continue
			}
			if _, err := tx.CW.Companies.Upsert(ctx, &models.Company{ID: c.ID, Name: c.Name}); err != nil {
				ph.step(fmt.Errorf("upserting company %d (%s): %w", c.ID, c.Name, err))
				continue
			}
			ph.step(nil)
		}
		return nil
	})
}

// SyncContacts refreshes the contacts ticketbot has stored, the same way SyncCompanies does. A
// contact that moved to a company ticketbot has not stored brings that company in.
func (s *Service) SyncContacts(ctx context.Context, ph *phase) error {
	stored, err := s.CW.Contacts.List(ctx)
	if err != nil {
		return fmt.Errorf("listing contacts from store: %w", err)
	}
	ids := liveIDs(stored, func(c *models.Contact) (int, bool) { return c.ID, c.Deleted })
	ph.fetching(fmt.Sprintf("Fetching %d stored contacts from ConnectWise", len(ids)))

	found := make(map[int]psa.Contact, len(ids))
	for batch := range slices.Chunk(ids, idBatch) {
		cwc, err := s.CW.CWClient.ListContacts(ctx, map[string]string{
			"conditions": idCondition(batch),
			"fields":     "id,firstName,lastName,company/id",
			"pageSize":   "1000",
		})
		if err != nil {
			return fmt.Errorf("listing contacts from connectwise: %w", err)
		}
		for _, c := range cwc {
			found[c.ID] = c
		}
	}

	return s.inTx(ctx, func(tx *Service) error {
		ph.counting("Syncing contacts", len(ids))
		for _, id := range ids {
			if err := ctx.Err(); err != nil {
				return err
			}
			c, ok := found[id]
			if !ok {
				if err := tx.CW.Contacts.SoftDelete(ctx, id); err != nil {
					ph.step(fmt.Errorf("soft deleting contact %d: %w", id, err))
					continue
				}
				ph.step(nil)
				continue
			}
			var companyID *int
			if c.Company.ID != 0 {
				if err := tx.companyStored(ctx, c.Company.ID); err != nil {
					ph.step(fmt.Errorf("storing company %d for contact %d: %w", c.Company.ID, c.ID, err))
					continue
				}
				companyID = &c.Company.ID
			}
			var last *string
			if c.LastName != "" {
				last = &c.LastName
			}
			if _, err := tx.CW.Contacts.Upsert(ctx, &models.Contact{ID: c.ID, FirstName: c.FirstName, LastName: last, CompanyID: companyID}); err != nil {
				ph.step(fmt.Errorf("upserting contact %d: %w", c.ID, err))
				continue
			}
			ph.step(nil)
		}
		return nil
	})
}

// companyStored makes sure the company a contact points at is stored, fetching it only when it
// is missing. The companies phase, which runs first, keeps the stored ones current.
func (s *Service) companyStored(ctx context.Context, id int) error {
	if _, err := s.CW.Companies.Get(ctx, id); err == nil {
		return nil
	} else if !errors.Is(err, models.ErrCompanyNotFound) {
		return err
	}
	_, err := s.CW.EnsureCompany(ctx, id)
	return err
}

// inTx runs fn against a copy of the service bound to one transaction and commits when fn
// returns nil.
func (s *Service) inTx(ctx context.Context, fn func(tx *Service) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}
	// not ctx: after a cancel the rollback still has to reach the database
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(s.withTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}
	return nil
}

func liveIDs[T any](items []T, idOf func(T) (int, bool)) []int {
	var ids []int
	for _, it := range items {
		if id, deleted := idOf(it); !deleted {
			ids = append(ids, id)
		}
	}
	return ids
}

func idCondition(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return "id in (" + strings.Join(parts, ",") + ")"
}
