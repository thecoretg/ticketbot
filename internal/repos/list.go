package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

// ListRepository stores admin-defined lists and their members.
type ListRepository interface {
	WithTx(tx pgx.Tx) ListRepository
	List(ctx context.Context) ([]*models.List, error)
	// Get returns models.ErrListNotFound for an unknown id.
	Get(ctx context.Context, id int) (*models.List, error)
	// Insert returns models.ErrListNameTaken when the name is already used (case-insensitive).
	Insert(ctx context.Context, l *models.List) (*models.List, error)
	// Update changes name and description only. ErrListNotFound / ErrListNameTaken as above.
	Update(ctx context.Context, l *models.List) (*models.List, error)
	Delete(ctx context.Context, id int) error
	// Items returns members with ItemID and AddedOn set; labels are resolved by the service.
	Items(ctx context.Context, listID int) ([]*models.ListItem, error)
	// AddItem reports whether the item was newly added (false when it was already a member).
	AddItem(ctx context.Context, listID, itemID int) (bool, error)
	// RemoveItem returns models.ErrListItemNotFound when the item was not a member.
	RemoveItem(ctx context.Context, listID, itemID int) error
	AllMemberships(ctx context.Context) ([]models.ListMembership, error)
}
