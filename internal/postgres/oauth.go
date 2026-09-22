package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
)

type OAuthRepo struct {
	queries *db.Queries
}

func NewOAuthRepo(pool *pgxpool.Pool) *OAuthRepo {
	return &OAuthRepo{queries: db.New(pool)}
}

func (r *OAuthRepo) CreateClient(ctx context.Context, c *models.OAuthClient) (*models.OAuthClient, error) {
	d, err := r.queries.CreateOAuthClient(ctx, db.CreateOAuthClientParams{
		ID: c.ID, Name: c.Name, RedirectUris: c.RedirectURIs, ExpiresAt: c.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return oauthClientFromPG(d), nil
}

func (r *OAuthRepo) GetClient(ctx context.Context, id string) (*models.OAuthClient, error) {
	d, err := r.queries.GetOAuthClient(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrOAuthClientNotFound
		}
		return nil, err
	}
	return oauthClientFromPG(d), nil
}

func (r *OAuthRepo) UpsertGrant(ctx context.Context, userID int, clientID string, scopes []string) (*models.OAuthGrant, error) {
	d, err := r.queries.UpsertOAuthGrant(ctx, db.UpsertOAuthGrantParams{UserID: userID, ClientID: clientID, Scopes: scopes})
	if err != nil {
		return nil, err
	}
	if err := r.queries.ClearOAuthClientExpiry(ctx, clientID); err != nil {
		return nil, err
	}
	return oauthGrantFromPG(d, ""), nil
}

func (r *OAuthRepo) GetGrant(ctx context.Context, id int) (*models.OAuthGrant, error) {
	d, err := r.queries.GetOAuthGrant(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrOAuthGrantNotFound
		}
		return nil, err
	}
	return oauthGrantFromPG(d, ""), nil
}

func (r *OAuthRepo) ListGrantsByUser(ctx context.Context, userID int) ([]*models.OAuthGrant, error) {
	rows, err := r.queries.ListOAuthGrantsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*models.OAuthGrant, 0, len(rows))
	for _, d := range rows {
		out = append(out, &models.OAuthGrant{
			ID: d.ID, UserID: d.UserID, ClientID: d.ClientID, ClientName: d.ClientName,
			Scopes: d.Scopes, CreatedOn: d.CreatedOn, LastUsedAt: d.LastUsedAt,
		})
	}
	return out, nil
}

func (r *OAuthRepo) DeleteGrant(ctx context.Context, id int) error {
	n, err := r.queries.DeleteOAuthGrant(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return models.ErrOAuthGrantNotFound
	}
	return nil
}

func (r *OAuthRepo) TouchGrant(ctx context.Context, id int) error {
	return r.queries.TouchOAuthGrant(ctx, id)
}

func (r *OAuthRepo) CreateCode(ctx context.Context, c *models.OAuthCode) error {
	return r.queries.CreateOAuthCode(ctx, db.CreateOAuthCodeParams{
		CodeHash: c.CodeHash, GrantID: c.GrantID, RedirectUri: c.RedirectURI,
		CodeChallenge: c.CodeChallenge, ExpiresAt: c.ExpiresAt,
	})
}

func (r *OAuthRepo) TakeCode(ctx context.Context, codeHash []byte) (*models.OAuthCode, error) {
	d, err := r.queries.TakeOAuthCode(ctx, codeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrOAuthCodeNotFound
		}
		return nil, err
	}
	return &models.OAuthCode{
		CodeHash: d.CodeHash, GrantID: d.GrantID, RedirectURI: d.RedirectUri,
		CodeChallenge: d.CodeChallenge, ExpiresAt: d.ExpiresAt,
	}, nil
}

func (r *OAuthRepo) CreateToken(ctx context.Context, t *models.OAuthToken) (*models.OAuthToken, error) {
	d, err := r.queries.CreateOAuthToken(ctx, db.CreateOAuthTokenParams{
		GrantID: t.GrantID, Kind: string(t.Kind), TokenHash: t.TokenHash, ExpiresAt: t.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return oauthTokenFromPG(d), nil
}

func (r *OAuthRepo) GetToken(ctx context.Context, tokenHash []byte) (*models.OAuthToken, error) {
	d, err := r.queries.GetOAuthToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrOAuthTokenNotFound
		}
		return nil, err
	}
	return oauthTokenFromPG(d), nil
}

func (r *OAuthRepo) MarkTokenUsed(ctx context.Context, id int) error {
	return r.queries.MarkOAuthTokenUsed(ctx, id)
}

func (r *OAuthRepo) ResolveAccessToken(ctx context.Context, tokenHash []byte) (*models.OAuthAccess, error) {
	d, err := r.queries.ResolveOAuthAccessToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrOAuthTokenNotFound
		}
		return nil, err
	}
	return &models.OAuthAccess{GrantID: d.GrantID, UserID: d.UserID, Scopes: d.Scopes, ExpiresAt: d.ExpiresAt}, nil
}

func (r *OAuthRepo) DeleteExpired(ctx context.Context, now time.Time) (models.OAuthPurgeCounts, error) {
	var c models.OAuthPurgeCounts
	var err error
	if c.Codes, err = r.queries.DeleteExpiredOAuthCodes(ctx, now); err != nil {
		return c, err
	}
	if c.Tokens, err = r.queries.DeleteExpiredOAuthTokens(ctx, now); err != nil {
		return c, err
	}
	if c.Clients, err = r.queries.DeleteExpiredOAuthClients(ctx, now); err != nil {
		return c, err
	}
	return c, nil
}

func oauthClientFromPG(d *db.OauthClient) *models.OAuthClient {
	return &models.OAuthClient{ID: d.ID, Name: d.Name, RedirectURIs: d.RedirectUris, CreatedOn: d.CreatedOn, ExpiresAt: d.ExpiresAt}
}

func oauthGrantFromPG(d *db.OauthGrant, clientName string) *models.OAuthGrant {
	return &models.OAuthGrant{
		ID: d.ID, UserID: d.UserID, ClientID: d.ClientID, ClientName: clientName,
		Scopes: d.Scopes, CreatedOn: d.CreatedOn, LastUsedAt: d.LastUsedAt,
	}
}

func oauthTokenFromPG(d *db.OauthToken) *models.OAuthToken {
	return &models.OAuthToken{
		ID: d.ID, GrantID: d.GrantID, Kind: models.OAuthTokenKind(d.Kind), TokenHash: d.TokenHash,
		ExpiresAt: d.ExpiresAt, CreatedOn: d.CreatedOn, UsedAt: d.UsedAt,
	}
}
