package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
)

type TOTPRecoveryRepo struct {
	queries *db.Queries
}

func NewTOTPRecoveryRepo(pool *pgxpool.Pool) *TOTPRecoveryRepo {
	return &TOTPRecoveryRepo{queries: db.New(pool)}
}

func (r *TOTPRecoveryRepo) Insert(ctx context.Context, userID int, codeHash []byte) error {
	return r.queries.InsertTOTPRecoveryCode(ctx, db.InsertTOTPRecoveryCodeParams{
		UserID:   userID,
		CodeHash: codeHash,
	})
}

func (r *TOTPRecoveryRepo) GetUnusedByHash(ctx context.Context, userID int, codeHash []byte) (*models.TOTPRecoveryCode, error) {
	d, err := r.queries.GetUnusedTOTPRecoveryCodeByHash(ctx, db.GetUnusedTOTPRecoveryCodeByHashParams{
		UserID:   userID,
		CodeHash: codeHash,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTOTPPendingNotFound
		}
		return nil, err
	}

	return &models.TOTPRecoveryCode{
		ID:       d.ID,
		UserID:   d.UserID,
		CodeHash: d.CodeHash,
		Used:     d.Used,
	}, nil
}

func (r *TOTPRecoveryRepo) MarkUsed(ctx context.Context, id int) error {
	return r.queries.MarkTOTPRecoveryCodeUsed(ctx, id)
}

func (r *TOTPRecoveryRepo) DeleteAll(ctx context.Context, userID int) error {
	return r.queries.DeleteTOTPRecoveryCodes(ctx, userID)
}
