package sdk

import (
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

func (c *Client) ListWorkflows() ([]models.Workflow, error) {
	return GetMany[models.Workflow](c, "workflows", nil)
}

func (c *Client) GetWorkflow(id int) (*models.Workflow, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}
	return GetOne[models.Workflow](c, fmt.Sprintf("workflows/%d", id), nil)
}

func (c *Client) GetWorkflowByBoard(boardID int) (*models.Workflow, error) {
	if boardID == 0 {
		return nil, errors.New("no board id provided")
	}
	return GetOne[models.Workflow](c, fmt.Sprintf("workflows/board/%d", boardID), nil)
}

func (c *Client) CreateWorkflow(payload *models.Workflow) (*models.Workflow, error) {
	w := &models.Workflow{}
	if err := c.Post("workflows", payload, w); err != nil {
		return nil, fmt.Errorf("posting to server: %w", err)
	}
	return w, nil
}

// ReplaceWorkflow overwrites the whole workflow document (rules included).
func (c *Client) ReplaceWorkflow(id int, payload *models.Workflow) (*models.Workflow, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}
	w := &models.Workflow{}
	if err := c.Put(fmt.Sprintf("workflows/%d", id), payload, w); err != nil {
		return nil, fmt.Errorf("sending update request: %w", err)
	}
	return w, nil
}

func (c *Client) DeleteWorkflow(id int) error {
	if id == 0 {
		return errors.New("no id provided")
	}
	return c.Delete(fmt.Sprintf("workflows/%d", id))
}

type ConditionCheck struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
	Pos   *int   `json:"pos,omitempty"`
}

func (c *Client) ValidateCondition(condition string) (*ConditionCheck, error) {
	out := &ConditionCheck{}
	if err := c.Post("workflows/validate-condition", map[string]string{"condition": condition}, out); err != nil {
		return nil, err
	}
	return out, nil
}

type ConditionEvaluation struct {
	Matches  bool           `json:"matches"`
	Source   string         `json:"source"`
	Document map[string]any `json:"document"`
}

func (c *Client) EvaluateCondition(condition string, ticketID int) (*ConditionEvaluation, error) {
	out := &ConditionEvaluation{}
	body := map[string]any{"condition": condition, "ticket_id": ticketID}
	if err := c.Post("workflows/evaluate-condition", body, out); err != nil {
		return nil, err
	}
	return out, nil
}
