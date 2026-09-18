package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type ListRepo struct {
	queries *db.Queries
}

func NewListRepo(pool *pgxpool.Pool) *ListRepo {
	return &ListRepo{queries: db.New(pool)}
}

func (p *ListRepo) WithTx(tx pgx.Tx) repos.ListRepository {
	return &ListRepo{queries: db.New(tx)}
}

func (p *ListRepo) List(ctx context.Context) ([]*models.List, error) {
	rows, err := p.queries.ListLists(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*models.List, 0, len(rows))
	for _, r := range rows {
		out = append(out, listFromParts(r.ID, r.Name, r.ItemType, r.Description, int(r.ItemCount), r.CreatedOn, r.UpdatedOn))
	}

	return out, nil
}

func (p *ListRepo) Get(ctx context.Context, id int) (*models.List, error) {
	r, err := p.queries.GetList(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrListNotFound
		}
		return nil, err
	}

	return listFromParts(r.ID, r.Name, r.ItemType, r.Description, int(r.ItemCount), r.CreatedOn, r.UpdatedOn), nil
}

func (p *ListRepo) Insert(ctx context.Context, l *models.List) (*models.List, error) {
	r, err := p.queries.InsertList(ctx, db.InsertListParams{Name: l.Name, ItemType: string(l.ItemType), Description: l.Description})
	if err != nil {
		return nil, listWriteError(err)
	}

	return listFromPG(r, 0), nil
}

func (p *ListRepo) Update(ctx context.Context, l *models.List) (*models.List, error) {
	r, err := p.queries.UpdateList(ctx, db.UpdateListParams{ID: l.ID, Name: l.Name, Description: l.Description})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrListNotFound
		}
		return nil, listWriteError(err)
	}

	return listFromPG(r, l.ItemCount), nil
}

func (p *ListRepo) Delete(ctx context.Context, id int) error {
	n, err := p.queries.DeleteList(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return models.ErrListNotFound
	}
	return nil
}

func (p *ListRepo) Items(ctx context.Context, listID int) ([]*models.ListItem, error) {
	rows, err := p.queries.ListListItems(ctx, listID)
	if err != nil {
		return nil, err
	}

	out := make([]*models.ListItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, &models.ListItem{ListID: r.ListID, ItemID: r.ItemID, AddedOn: r.AddedOn})
	}

	return out, nil
}

func (p *ListRepo) AddItem(ctx context.Context, listID, itemID int) (bool, error) {
	n, err := p.queries.InsertListItem(ctx, db.InsertListItemParams{ListID: listID, ItemID: itemID})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return false, models.ErrListNotFound
		}
		return false, err
	}
	if n > 0 {
		if err := p.queries.TouchList(ctx, listID); err != nil {
			return true, err
		}
	}
	return n > 0, nil
}

func (p *ListRepo) RemoveItem(ctx context.Context, listID, itemID int) error {
	n, err := p.queries.DeleteListItem(ctx, db.DeleteListItemParams{ListID: listID, ItemID: itemID})
	if err != nil {
		return err
	}
	if n == 0 {
		return models.ErrListItemNotFound
	}
	return p.queries.TouchList(ctx, listID)
}

func (p *ListRepo) AllMemberships(ctx context.Context) ([]models.ListMembership, error) {
	rows, err := p.queries.ListAllListItems(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]models.ListMembership, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.ListMembership{ListID: r.ListID, ItemID: r.ItemID})
	}

	return out, nil
}

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// listWriteError maps the unique index on LOWER(name) to ErrListNameTaken.
func listWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return models.ErrListNameTaken
	}
	return err
}

func listFromPG(r *db.List, count int) *models.List {
	return listFromParts(r.ID, r.Name, r.ItemType, r.Description, count, r.CreatedOn, r.UpdatedOn)
}

func listFromParts(id int, name, itemType, description string, count int, created, updated time.Time) *models.List {
	return &models.List{
		ID:          id,
		Name:        name,
		ItemType:    models.ListItemType(itemType),
		Description: description,
		ItemCount:   count,
		CreatedOn:   created,
		UpdatedOn:   updated,
	}
}
