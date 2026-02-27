package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
)

type SessionRepo struct {
	queries *db.Queries
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{queries: db.New(pool)}
}

func (r *SessionRepo) Create(ctx context.Context, s *models.Session) (*models.Session, error) {
	d, err := r.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    s.UserID,
		TokenHash: s.TokenHash,
		ExpiresAt: s.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	return sessionFromPG(d), nil
}

func (r *SessionRepo) GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.Session, error) {
	d, err := r.queries.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrSessionNotFound
		}
		return nil, err
	}

	return sessionFromPG(d), nil
}

func (r *SessionRepo) Delete(ctx context.Context, id int) error {
	return r.queries.DeleteSession(ctx, id)
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	return r.queries.DeleteExpiredSessions(ctx)
}

func sessionFromPG(d *db.Session) *models.Session {
	return &models.Session{
		ID:        d.ID,
		UserID:    d.UserID,
		TokenHash: d.TokenHash,
		ExpiresAt: d.ExpiresAt,
		CreatedOn: d.CreatedOn,
	}
}
