package transformer

import (
	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Cfg      *models.Config
	CWClient *psa.Client
	Rules    repos.TransformerRuleRepository
	Runs     repos.TransformerRunRepository
	registry map[string]Transformer
}

type SvcParams struct {
	Cfg      *models.Config
	CWClient *psa.Client
	Rules    repos.TransformerRuleRepository
	Runs     repos.TransformerRunRepository
}

func New(p SvcParams) *Service {
	return &Service{
		Cfg:      p.Cfg,
		CWClient: p.CWClient,
		Rules:    p.Rules,
		Runs:     p.Runs,
		registry: newRegistry(),
	}
}

func (s *Service) exec() *Exec {
	return &Exec{
		CW:                  s.CWClient,
		BotMemberIdentifier: s.Cfg.CwBotMemberIdentifier,
	}
}
