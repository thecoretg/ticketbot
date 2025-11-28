package cwsvc

import (
	"context"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (s *Service) ListBoards(ctx context.Context) ([]models.Board, error) {
	return s.Boards.List(ctx)
}

func (s *Service) GetBoard(ctx context.Context, id int) (models.Board, error) {
	return s.Boards.Get(ctx, id)
}
