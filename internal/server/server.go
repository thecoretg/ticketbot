package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/authsvc"
	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/lists"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/syncsvc"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/internal/service/user"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/internal/service/webhooks"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type App struct {
	Creds                   *Creds
	TestFlags               *TestFlags
	Stores                  *repos.AllRepos
	CWClient                *psa.Client
	MessageSender           repos.MessageSender
	Pool                    *pgxpool.Pool
	Config                  *models.Config
	Svc                     *Services
	CurrentMigrationVersion int64
	LogBuffer               *logging.BufferHandler
}

type Services struct {
	Auth      *authsvc.Service
	Config    *config.Service
	User      *user.Service
	CW        *cwsvc.Service
	Hooks     *webhooks.Service
	Webex     *webexsvc.Service
	Sync      *syncsvc.Service
	Notifier  *notifier.Service
	Ticketbot *ticketbot.Service
	Workflow  *workflow.Service
	Lists     *lists.Service
}

const defaultStoreTTL = int64(900)

func NewApp(ctx context.Context, migVersion int64, level *slog.LevelVar, logBuf *logging.BufferHandler) (*App, *logging.Persister, error) {
	cr := getCreds()
	tf := getTestFlags()
	if err := cr.validate(tf); err != nil {
		return nil, nil, fmt.Errorf("validating credentials: %w", err)
	}

	ttl := defaultStoreTTL
	if tf.StoreTTLSeconds != 0 {
		ttl = tf.StoreTTLSeconds
	}
	slog.Info("using TTL", "ttl", ttl)

	cw, err := psa.NewClient(ctx, *cr.CWCreds)
	if err != nil {
		return nil, nil, fmt.Errorf("creating connectwise client: %w", err)
	}

	ms, err := makeMessageSender(ctx, tf.MockWebex, cr.WebexAPISecret)
	if err != nil {
		return nil, nil, fmt.Errorf("creating message sender: %w", err)
	}

	s, err := CreateStores(ctx, cr, migVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing stores: %w", err)
	}
	r := s.Repos

	cfg, err := getStartupConfig(ctx, r.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("getting initial config: %w", err)
	}

	cws := cwsvc.New(s.Pool, r.CW, r.TicketEvents, cw, cr.CWCreds.CompanyID, ttl)
	ws := webexsvc.New(s.Pool, r.WebexRecipients, ms)

	nr := notifier.SvcParams{
		Cfg:           cfg,
		WebexSvc:      ws,
		Notifications: r.TicketNotifications,
		Forwards:      r.NotifierForwards,
		Pool:          s.Pool,
		MessageSender: ms,
		CWCompanyID:   cr.CWCreds.CompanyID,
	}

	ns := notifier.New(nr)
	cfgSvc := config.New(r.Config, cfg, level, logBuf)
	listSvc := lists.New(lists.Params{Lists: r.Lists, Companies: r.CW.Company, Contacts: r.CW.Contact, Workflows: r.Workflows})
	engine := workflow.NewEngine(cw)
	engine.Lists = listSvc
	tb := ticketbot.New(ticketbot.Params{
		Cfg:       cfg,
		ConfigSvc: cfgSvc,
		CW:        cws,
		Workflows: r.Workflows,
		Events:    r.TicketEvents,
		Engine:    engine,
		Notifier:  ns,
	})

	persister := logging.NewPersister(r.Logs, logBuf, cfg)

	return &App{
		Creds:         cr,
		Config:        cfg,
		TestFlags:     tf,
		Stores:        r,
		Pool:          s.Pool,
		CWClient:      cw,
		MessageSender: ms,
		LogBuffer:     logBuf,
		Svc: &Services{
			Auth:      authsvc.New(r.APIUser, r.Sessions, r.TOTPPending, r.TOTPRecovery, cfg),
			Config:    cfgSvc,
			User:      user.New(r.APIUser, r.APIKey),
			Hooks:     webhooks.New(cw, cr.RootURL),
			CW:        cws,
			Webex:     ws,
			Sync:      syncsvc.New(s.Pool, cws, ws, tb),
			Notifier:  ns,
			Ticketbot: tb,
			Workflow: workflow.New(workflow.Params{
				Workflows:  r.Workflows,
				Recipients: r.WebexRecipients,
				Boards:     r.CW.Board,
				Statuses:   r.CW.TicketStatus,
				Members:    r.CW.Member,
				Lists:      r.Lists,
			}),
			Lists: listSvc,
		},
	}, persister, nil
}
