package cwsvc

import (
	"context"

	"github.com/thecoretg/ticketbot/models"
)

func (s *Service) ListStatuses(ctx context.Context) ([]*models.TicketStatus, error) {
	return s.Statuses.List(ctx)
}

func (s *Service) ListStatusesByBoard(ctx context.Context, boardID int) ([]*models.TicketStatus, error) {
	return s.Statuses.ListByBoard(ctx, boardID)
}
