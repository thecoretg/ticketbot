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
		Event:        res.Event,
		Steps:        make([]models.StepPayload, 0, len(res.Steps)),
	}
	for _, st := range res.Steps {
		p.Steps = append(p.Steps, st.Payload())
	}

	return p
}

func notificationPayload(o notifier.Outcome) models.NotificationPayload {
	p := models.NotificationPayload{
		NodeID:        o.Step.NodeID,
		Title:         o.Step.Title,
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
