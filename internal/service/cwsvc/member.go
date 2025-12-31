package cwsvc

import (
	"context"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (s *Service) ListMembers(ctx context.Context) ([]*models.Member, error) {
	return s.Members.List(ctx)
}
