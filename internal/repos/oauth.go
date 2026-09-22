package repos

import (
	"context"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

// OAuthRepository stores the MCP server's OAuth clients, grants, codes and tokens.
type OAuthRepository interface {
	CreateClient(ctx context.Context, c *models.OAuthClient) (*models.OAuthClient, error)
	GetClient(ctx context.Context, id string) (*models.OAuthClient, error)

	// UpsertGrant records or refreshes the user's approval of a client and marks the client as
	// kept (clears its registration expiry).
	UpsertGrant(ctx context.Context, userID int, clientID string, scopes []string) (*models.OAuthGrant, error)
	GetGrant(ctx context.Context, id int) (*models.OAuthGrant, error)
	ListGrantsByUser(ctx context.Context, userID int) ([]*models.OAuthGrant, error)
	DeleteGrant(ctx context.Context, id int) error
	TouchGrant(ctx context.Context, id int) error

	CreateCode(ctx context.Context, c *models.OAuthCode) error
	// TakeCode returns and deletes the code in one step so it can be exchanged once.
	TakeCode(ctx context.Context, codeHash []byte) (*models.OAuthCode, error)

	CreateToken(ctx context.Context, t *models.OAuthToken) (*models.OAuthToken, error)
	GetToken(ctx context.Context, tokenHash []byte) (*models.OAuthToken, error)
	MarkTokenUsed(ctx context.Context, id int) error
	ResolveAccessToken(ctx context.Context, tokenHash []byte) (*models.OAuthAccess, error)

	DeleteExpired(ctx context.Context, now time.Time) (models.OAuthPurgeCounts, error)
}
