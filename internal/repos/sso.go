package repos

import (
	"context"

	"github.com/thecoretg/tctg-go/entra"
	"github.com/thecoretg/ticketbot/models"
)

// SSOStore persists Entra sign-in flows and sessions. It is the Postgres-backed replacement
// for entra.MemoryStore, so sessions survive a restart.
type SSOStore interface {
	entra.SessionStore
	entra.StateStore
}

type SSORoleMappingRepository interface {
	List(ctx context.Context) ([]*models.SSORoleMapping, error)
	Upsert(ctx context.Context, entraRole string, role models.Role) (*models.SSORoleMapping, error)
	Delete(ctx context.Context, id int) error
}
