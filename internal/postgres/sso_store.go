package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/tctg-go/entra"
	"github.com/thecoretg/ticketbot/internal/db"
)

// SSOStore implements repos.SSOStore. Keys are the digests entra hands us; the raw cookie never
// reaches the database. Expired rows are swept opportunistically on every Put, as
// entra.MemoryStore does, so no scheduler is needed.
type SSOStore struct {
	queries *db.Queries
}

func NewSSOStore(pool *pgxpool.Pool) *SSOStore {
	return &SSOStore{queries: db.New(pool)}
}

func (s *SSOStore) PutState(ctx context.Context, key string, fs entra.FlowState) error {
	if err := s.queries.DeleteExpiredSSOFlowStates(ctx); err != nil {
		return fmt.Errorf("sweep expired sso flow states: %w", err)
	}
	return s.queries.PutSSOFlowState(ctx, db.PutSSOFlowStateParams{
		Key:       key,
		State:     fs.State,
		Nonce:     fs.Nonce,
		Verifier:  fs.Verifier,
		Next:      fs.Next,
		ExpiresAt: fs.ExpiresAt,
	})
}

func (s *SSOStore) TakeState(ctx context.Context, key string) (entra.FlowState, bool, error) {
	d, err := s.queries.TakeSSOFlowState(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entra.FlowState{}, false, nil
		}
		return entra.FlowState{}, false, err
	}
	return entra.FlowState{
		State:     d.State,
		Nonce:     d.Nonce,
		Verifier:  d.Verifier,
		Next:      d.Next,
		ExpiresAt: d.ExpiresAt,
	}, true, nil
}

func (s *SSOStore) PutSession(ctx context.Context, key string, sess entra.Session) error {
	userID, err := parseUserID(sess.UserID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteExpiredSSOSessions(ctx); err != nil {
		return fmt.Errorf("sweep expired sso sessions: %w", err)
	}
	return s.queries.PutSSOSession(ctx, db.PutSSOSessionParams{
		Key:        key,
		UserID:     userID,
		CreatedOn:  sess.CreatedAt,
		LastSeenAt: sess.LastSeenAt,
		ExpiresAt:  sess.ExpiresAt,
	})
}

func (s *SSOStore) GetSession(ctx context.Context, key string) (entra.Session, bool, error) {
	d, err := s.queries.GetSSOSession(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entra.Session{}, false, nil
		}
		return entra.Session{}, false, err
	}
	return entra.Session{
		UserID:     strconv.Itoa(d.UserID),
		CreatedAt:  d.CreatedOn,
		LastSeenAt: d.LastSeenAt,
		ExpiresAt:  d.ExpiresAt,
	}, true, nil
}

func (s *SSOStore) TouchSession(ctx context.Context, key string, lastSeen, expires time.Time) error {
	return s.queries.TouchSSOSession(ctx, db.TouchSSOSessionParams{Key: key, LastSeenAt: lastSeen, ExpiresAt: expires})
}

func (s *SSOStore) DeleteSession(ctx context.Context, key string) error {
	return s.queries.DeleteSSOSession(ctx, key)
}

func (s *SSOStore) DeleteUserSessions(ctx context.Context, userID string) error {
	id, err := parseUserID(userID)
	if err != nil {
		return err
	}
	return s.queries.DeleteSSOSessionsByUser(ctx, id)
}

// parseUserID converts the opaque string entra carries back to the api_user primary key.
func parseUserID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("sso session user id %q is not an integer: %w", s, err)
	}
	return id, nil
}
