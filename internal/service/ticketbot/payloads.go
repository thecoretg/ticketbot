package ticketbot

import (
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

func workflowPayload(wf *models.Workflow, res *workflow.Result) models.WorkflowPayload {
	p := models.WorkflowPayload{
		WorkflowID:   &wf.ID,
		WorkflowName: wf.Name,
		Found:        true,
		Enabled:      true,
		Rules:        make([]models.RuleOutcomePayload, 0, len(res.Rules)),
	}

	for _, r := range res.Rules {
		rp := models.RuleOutcomePayload{
			RuleID:   r.Rule.RuleID,
			RuleName: r.Rule.RuleName,
			Matched:  r.Matched,
			Skipped:  r.Skipped,
			Stopped:  r.Stopped,
		}
		if r.Err != nil {
			rp.Error = r.Err.Error()
		}
		p.Rules = append(p.Rules, rp)
	}

	return p
}

func actionPayload(a workflow.ActionOutcome) models.ActionPayload {
	p := models.ActionPayload{
		RuleID:   a.Rule.RuleID,
		RuleName: a.Rule.RuleName,
		Index:    a.Index,
		Kind:     string(a.Kind),
		Result:   a.Result,
		Reason:   a.Reason,
		Output:   a.Output,
	}
	if a.Err != nil {
		p.Error = a.Err.Error()
	}

	return p
}

func notificationPayload(o notifier.Outcome) models.NotificationPayload {
	p := models.NotificationPayload{
		RuleID:        o.Rule.RuleID,
		RuleName:      o.Rule.RuleName,
		ForwardedFrom: o.ForwardedFrom,
		Result:        o.Result,
	}
	if o.Recipient != nil {
		p.RecipientID = o.Recipient.ID
		p.RecipientName = o.Recipient.Name
		p.RecipientType = string(o.Recipient.Type)
	}
	if o.Err != nil {
		p.Error = o.Err.Error()
	}
	if o.Notification != nil && o.Notification.ID != 0 {
		id := o.Notification.ID
		p.NotificationID = &id
	}

	return p
}
