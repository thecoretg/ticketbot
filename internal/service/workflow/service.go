package workflow

import (
	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Cfg         *models.Config
	CWClient    *psa.Client
	Workflows   repos.WorkflowRepository
	Runs        repos.WorkflowRunRepository
	Webex       repos.MessageSender
	Recips      repos.WebexRecipientRepository
	CWCompanyID string
	registry    map[string]ActionHandler
}

type SvcParams struct {
	Cfg         *models.Config
	CWClient    *psa.Client
	Workflows   repos.WorkflowRepository
	Runs        repos.WorkflowRunRepository
	Webex       repos.MessageSender
	Recips      repos.WebexRecipientRepository
	CWCompanyID string
}

func New(p SvcParams) *Service {
	return &Service{
		Cfg:         p.Cfg,
		CWClient:    p.CWClient,
		Workflows:   p.Workflows,
		Runs:        p.Runs,
		Webex:       p.Webex,
		Recips:      p.Recips,
		CWCompanyID: p.CWCompanyID,
		registry:    newRegistry(),
	}
}

// UpdateFields returns the catalog of fields a ticket_update action may target,
// for the admin panel's op builder.
func (s *Service) UpdateFields() []UpdateFieldInfo {
	return UpdateFields()
}

func (s *Service) exec(lastNote *psa.ServiceTicketNote) *Exec {
	return &Exec{
		CW:               s.CWClient,
		Webex:            s.Webex,
		Recips:           s.Recips,
		CWCompanyID:      s.CWCompanyID,
		MaxMessageLength: s.Cfg.MaxMessageLength,
		LastNote:         lastNote,
	}
}
