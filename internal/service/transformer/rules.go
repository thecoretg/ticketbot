package transformer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

var (
	ErrUnknownAction = errors.New("unknown transformer action")
	ErrInvalidConfig = errors.New("invalid transformer rule config")
)

func (s *Service) ListTransformerRules(ctx context.Context) ([]*models.TransformerRule, error) {
	return s.Rules.ListAll(ctx)
}

func (s *Service) GetTransformerRule(ctx context.Context, id int) (*models.TransformerRule, error) {
	return s.Rules.Get(ctx, id)
}

func (s *Service) AddTransformerRule(ctx context.Context, r *models.TransformerRule) (*models.TransformerRule, error) {
	if err := s.validateRule(r); err != nil {
		return nil, err
	}
	return s.Rules.Insert(ctx, r)
}

func (s *Service) UpdateTransformerRule(ctx context.Context, r *models.TransformerRule) (*models.TransformerRule, error) {
	if err := s.validateRule(r); err != nil {
		return nil, err
	}

	exists, err := s.Rules.Exists(ctx, r.ID)
	if err != nil {
		return nil, fmt.Errorf("checking if transformer rule exists: %w", err)
	}
	if !exists {
		return nil, models.ErrTransformerRuleNotFound
	}

	return s.Rules.Update(ctx, r)
}

func (s *Service) DeleteTransformerRule(ctx context.Context, id int) error {
	exists, err := s.Rules.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("checking if transformer rule exists: %w", err)
	}
	if !exists {
		return models.ErrTransformerRuleNotFound
	}
	return s.Rules.Delete(ctx, id)
}

// validateRule checks the action is known, the config parses into that action's
// params, and every templated field compiles — so bad rules are rejected at save
// time rather than failing silently when a webhook arrives.
func (s *Service) validateRule(r *models.TransformerRule) error {
	if r == nil {
		return errors.New("got nil transformer rule")
	}

	tf, ok := s.registry[r.Action]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownAction, r.Action)
	}

	switch r.ApplyOn {
	case models.TransformerApplyNew, models.TransformerApplyUpdated, models.TransformerApplyBoth:
	case "":
		r.ApplyOn = models.TransformerApplyBoth
	default:
		return fmt.Errorf("%w: invalid apply_on %q", ErrInvalidConfig, r.ApplyOn)
	}

	if err := validateConditions(r.Conditions); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	params := tf.NewParams()
	if err := json.Unmarshal(rawConfig(r.Config), params); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if err := validateTemplates(params); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	return nil
}
