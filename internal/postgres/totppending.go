package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
)

type TOTPPendingRepo struct {
	queries *db.Queries
}

func NewTOTPPendingRepo(pool *pgxpool.Pool) *TOTPPendingRepo {
	return &TOTPPendingRepo{queries: db.New(pool)}
}

func (r *TOTPPendingRepo) Create(ctx context.Context, p *models.TOTPPending) (*models.TOTPPending, error) {
	d, err := r.queries.CreateTOTPPending(ctx, db.CreateTOTPPendingParams{
		UserID:    p.UserID,
		TokenHash: p.TokenHash,
		ExpiresAt: p.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	return totpPendingFromPG(d), nil
}

func (r *TOTPPendingRepo) GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.TOTPPending, error) {
	d, err := r.queries.GetTOTPPendingByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTOTPPendingNotFound
		}
		return nil, err
	}

	return totpPendingFromPG(d), nil
}

func (r *TOTPPendingRepo) Delete(ctx context.Context, id int) error {
	return r.queries.DeleteTOTPPending(ctx, id)
}

func (r *TOTPPendingRepo) DeleteExpired(ctx context.Context) error {
	return r.queries.DeleteExpiredTOTPPendings(ctx)
}

func totpPendingFromPG(d *db.TotpPending) *models.TOTPPending {
	return &models.TOTPPending{
		ID:        d.ID,
		UserID:    d.UserID,
		TokenHash: d.TokenHash,
		ExpiresAt: d.ExpiresAt,
		CreatedOn: d.CreatedOn,
	}
}
