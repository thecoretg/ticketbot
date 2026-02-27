package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/ticketbot/internal/repos"
)

type BoardRepo struct {
	queries *db.Queries
}

func NewBoardRepo(pool *pgxpool.Pool) *BoardRepo {
	return &BoardRepo{
		queries: db.New(pool),
	}
}

func (p *BoardRepo) WithTx(tx pgx.Tx) repos.BoardRepository {
	return &BoardRepo{
		queries: db.New(tx),
	}
}

func (p *BoardRepo) List(ctx context.Context) ([]*models.Board, error) {
	dbs, err := p.queries.ListBoards(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.Board
	for _, d := range dbs {
		b = append(b, boardFromPG(d))
	}

	return b, nil
}

func (p *BoardRepo) Get(ctx context.Context, id int) (*models.Board, error) {
	d, err := p.queries.GetBoard(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrBoardNotFound
		}
		return nil, err
	}

	return boardFromPG(d), nil
}

func (p *BoardRepo) Upsert(ctx context.Context, b *models.Board) (*models.Board, error) {
	d, err := p.queries.UpsertBoard(ctx, boardToUpsertParams(b))
	if err != nil {
		return nil, err
	}

	return boardFromPG(d), nil
}

func (p *BoardRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteBoard(ctx, id)
}

func (p *BoardRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteBoard(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrBoardNotFound
		}
		return err
	}

	return nil
}

func boardToUpsertParams(b *models.Board) db.UpsertBoardParams {
	return db.UpsertBoardParams{
		ID:   b.ID,
		Name: b.Name,
	}
}

func boardFromPG(pg *db.CwBoard) *models.Board {
	return &models.Board{
		ID:        pg.ID,
		Name:      pg.Name,
		UpdatedOn: pg.UpdatedOn,
		AddedOn:   pg.AddedOn,
		Deleted:   pg.Deleted,
	}
}
