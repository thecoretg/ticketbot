package notifier

import (
	"context"
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
)

var ErrNotifierConflict = errors.New("notifier already exists with this board and webex recipient")

func (s *Service) ListNotifierRules(ctx context.Context) ([]*models.NotifierRuleFull, error) {
	return s.NotifierRules.ListAllFull(ctx)
}

func (s *Service) GetNotifierRule(ctx context.Context, id int) (*models.NotifierRule, error) {
	return s.NotifierRules.Get(ctx, id)
}

func (s *Service) DeleteNotifierRule(ctx context.Context, id int) error {
	exists, err := s.NotifierRules.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("checking if notifier exists: %w", err)
	}

	if !exists {
		return models.ErrNotifierNotFound
	}

	return s.NotifierRules.Delete(ctx, id)
}

func (s *Service) AddNotifierRule(ctx context.Context, nr *models.NotifierRule) (*models.NotifierRule, error) {
	if nr == nil {
		return nil, errors.New("got nil notifier rule")
	}

	exists, err := s.NotifierRules.ExistsByBoardAndRecipient(ctx, nr.CwBoardID, nr.WebexRecipientID)
	if err != nil {
		return nil, fmt.Errorf("checking if notifier rule exists: %w", err)
	}

	if exists {
		return nil, ErrNotifierConflict
	}

	n, err := s.NotifierRules.Insert(ctx, nr)
	if err != nil {
		return nil, fmt.Errorf("adding notifier rule: %w", err)
	}

	return n, nil
}
