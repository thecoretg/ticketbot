package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

var (
	ErrUnknownAction = errors.New("unknown workflow action")
	ErrInvalidConfig = errors.New("invalid workflow config")
)

func (s *Service) ListWorkflows(ctx context.Context) ([]*models.Workflow, error) {
	return s.Workflows.ListAll(ctx)
}

func (s *Service) GetWorkflow(ctx context.Context, id int) (*models.Workflow, error) {
	return s.Workflows.Get(ctx, id)
}

func (s *Service) AddWorkflow(ctx context.Context, w *models.Workflow) (*models.Workflow, error) {
	if err := s.validateWorkflow(w); err != nil {
		return nil, err
	}
	return s.Workflows.Insert(ctx, w)
}

func (s *Service) UpdateWorkflow(ctx context.Context, w *models.Workflow) (*models.Workflow, error) {
	if err := s.validateWorkflow(w); err != nil {
		return nil, err
	}

	exists, err := s.Workflows.Exists(ctx, w.ID)
	if err != nil {
		return nil, fmt.Errorf("checking if workflow exists: %w", err)
	}
	if !exists {
		return nil, models.ErrWorkflowNotFound
	}

	return s.Workflows.Update(ctx, w)
}

func (s *Service) DeleteWorkflow(ctx context.Context, id int) error {
	exists, err := s.Workflows.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("checking if workflow exists: %w", err)
	}
	if !exists {
		return models.ErrWorkflowNotFound
	}
	return s.Workflows.Delete(ctx, id)
}

// validateWorkflow checks the board is set, the on-ticket-action is valid, the
// condition tree references known fields/operators, and every action is a known
// type whose config parses and whose templated fields compile — so bad workflows
// are rejected at save time rather than failing silently when a webhook arrives.
func (s *Service) validateWorkflow(w *models.Workflow) error {
	if w == nil {
		return errors.New("got nil workflow")
	}

	if w.CwBoardID == 0 {
		return fmt.Errorf("%w: a board is required", ErrInvalidConfig)
	}

	switch w.OnTicketAction {
	case models.WorkflowOnNew, models.WorkflowOnUpdated, models.WorkflowOnBoth:
	case "":
		w.OnTicketAction = models.WorkflowOnBoth
	default:
		return fmt.Errorf("%w: invalid on_ticket_action %q", ErrInvalidConfig, w.OnTicketAction)
	}

	if err := validateGroup(w.Root, 0); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	if len(w.Actions) == 0 {
		return fmt.Errorf("%w: at least one action is required", ErrInvalidConfig)
	}

	for i, a := range w.Actions {
		handler, ok := s.registry[a.Type]
		if !ok {
			return fmt.Errorf("%w: action %d: %q", ErrUnknownAction, i+1, a.Type)
		}

		params := handler.NewParams()
		if err := json.Unmarshal(rawConfig(a.Config), params); err != nil {
			return fmt.Errorf("%w: action %d (%s): %v", ErrInvalidConfig, i+1, a.Type, err)
		}
		if err := validateParamsTemplates(params); err != nil {
			return fmt.Errorf("%w: action %d (%s): %v", ErrInvalidConfig, i+1, a.Type, err)
		}

		switch pp := params.(type) {
		case *TicketUpdateParams:
			// The workflow always has a board, so the status→board dependency is satisfied.
			if err := pp.checkDependencies(true); err != nil {
				return fmt.Errorf("%w: action %d (%s): %v", ErrInvalidConfig, i+1, a.Type, err)
			}
		case *SendMessageParams:
			if pp.RecipientID == 0 {
				return fmt.Errorf("%w: action %d (%s): a recipient is required", ErrInvalidConfig, i+1, a.Type)
			}
		case *AddResourceParams:
			if pp.MemberIdentifier == "" {
				return fmt.Errorf("%w: action %d (%s): a member is required", ErrInvalidConfig, i+1, a.Type)
			}
		case *AddEmailCcParams:
			if pp.Email == "" {
				return fmt.Errorf("%w: action %d (%s): an email address is required", ErrInvalidConfig, i+1, a.Type)
			}
		}
	}

	return nil
}
