package repos

import (
	"context"

	"github.com/thecoretg/ticketbot/models"
)

type TOTPPendingRepository interface {
	Create(ctx context.Context, p *models.TOTPPending) (*models.TOTPPending, error)
	GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.TOTPPending, error)
	Delete(ctx context.Context, id int) error
	DeleteExpired(ctx context.Context) error
}

type TOTPRecoveryRepository interface {
	Insert(ctx context.Context, userID int, codeHash []byte) error
	GetUnusedByHash(ctx context.Context, userID int, codeHash []byte) (*models.TOTPRecoveryCode, error)
	MarkUsed(ctx context.Context, id int) error
	DeleteAll(ctx context.Context, userID int) error
}
