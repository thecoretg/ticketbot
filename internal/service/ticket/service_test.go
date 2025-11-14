package ticket

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNewService(t *testing.T) {
	ctx := context.Background()
	pgdsn := os.Getenv("POSTGRES_DSN")
	cwPubKey := os.Getenv("CW_PUB_KEY")
	cwPrivKey := os.Getenv("CW_PRIV_KEY")
	cwClientID := os.Getenv("CW_CLIENT_ID")
	cwCompanyID := os.Getenv("CW_COMPANY_ID")

	pool, err := pgxpool.New(ctx, pgdsn)
	if err != nil {
		t.Errorf("connecting to pgx pool: %v", err)
	}
}
