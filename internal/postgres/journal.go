package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type TicketJournalRepo struct {
	queries *db.Queries
}

func NewTicketJournalRepo(pool *pgxpool.Pool) *TicketJournalRepo {
	return &TicketJournalRepo{queries: db.New(pool)}
}

func (p *TicketJournalRepo) WithTx(tx pgx.Tx) repos.TicketJournalRepository {
	return &TicketJournalRepo{queries: db.New(tx)}
}

func (p *TicketJournalRepo) Get(ctx context.Context, ticketID int) (*models.TicketJournal, error) {
	d, err := p.queries.GetTicketJournal(ctx, ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTicketJournalNotFound
		}
		return nil, err
	}
	return ticketJournalFromPG(d), nil
}

func (p *TicketJournalRepo) ListSummaries(ctx context.Context, limit int) ([]*models.TicketJournal, error) {
	rows, err := p.queries.ListTicketJournalSummaries(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	out := make([]*models.TicketJournal, 0, len(rows))
	for _, r := range rows {
		out = append(out, &models.TicketJournal{
			TicketID:      r.TicketID,
			Summary:       r.Summary,
			BoardName:     r.BoardName,
			CompanyName:   r.CompanyName,
			ContactName:   r.ContactName,
			StatusName:    r.StatusName,
			OwnerName:     r.OwnerName,
			TypeName:      r.TypeName,
			SubtypeName:   r.SubtypeName,
			ItemName:      r.ItemName,
			ResourceNames: r.ResourceNames,
			LastTrigger:   r.LastTrigger,
			LastOutcome:   r.LastOutcome,
			HadError:      r.HadError,
			FirstSeen:     r.FirstSeen,
			LastRun:       r.LastRun,
		})
	}
	return out, nil
}

func (p *TicketJournalRepo) Upsert(ctx context.Context, j *models.TicketJournal) (*models.TicketJournal, error) {
	d, err := p.queries.UpsertTicketJournal(ctx, db.UpsertTicketJournalParams{
		TicketID:      j.TicketID,
		Summary:       j.Summary,
		BoardName:     j.BoardName,
		CompanyName:   j.CompanyName,
		ContactName:   j.ContactName,
		StatusName:    j.StatusName,
		OwnerName:     j.OwnerName,
		TypeName:      j.TypeName,
		SubtypeName:   j.SubtypeName,
		ItemName:      j.ItemName,
		ResourceNames: j.ResourceNames,
		LastTrigger:   j.LastTrigger,
		LastOutcome:   j.LastOutcome,
		HadError:      j.HadError,
		FirstSeen:     j.FirstSeen,
		LastRun:       j.LastRun,
		Runs:          runsBytes(j.Runs),
	})
	if err != nil {
		return nil, err
	}
	return ticketJournalFromPG(d), nil
}

func (p *TicketJournalRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return p.queries.DeleteTicketJournalsOlderThan(ctx, before)
}

func ticketJournalFromPG(pg *db.TicketJournal) *models.TicketJournal {
	return &models.TicketJournal{
		TicketID:      pg.TicketID,
		Summary:       pg.Summary,
		BoardName:     pg.BoardName,
		CompanyName:   pg.CompanyName,
		ContactName:   pg.ContactName,
		StatusName:    pg.StatusName,
		OwnerName:     pg.OwnerName,
		TypeName:      pg.TypeName,
		SubtypeName:   pg.SubtypeName,
		ItemName:      pg.ItemName,
		ResourceNames: pg.ResourceNames,
		LastTrigger:   pg.LastTrigger,
		LastOutcome:   pg.LastOutcome,
		HadError:      pg.HadError,
		FirstSeen:     pg.FirstSeen,
		LastRun:       pg.LastRun,
		Runs:          runsFromBytes(pg.Runs),
	}
}

func runsBytes(runs []models.TicketRun) []byte {
	if len(runs) == 0 {
		return []byte("[]")
	}
	b, err := json.Marshal(runs)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func runsFromBytes(b []byte) []models.TicketRun {
	if len(b) == 0 {
		return nil
	}
	var runs []models.TicketRun
	if err := json.Unmarshal(b, &runs); err != nil {
		return nil
	}
	return runs
}
