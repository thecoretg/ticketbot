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
	"github.com/thecoretg/ticketbot/internal/service/journal"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/syncsvc"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/internal/service/user"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/internal/service/webhooks"
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
	Auth        *authsvc.Service
	Config      *config.Service
	User        *user.Service
	CW          *cwsvc.Service
	Hooks       *webhooks.Service
	Webex       *webexsvc.Service
	Sync        *syncsvc.Service
	Notifier    *notifier.Service
	Workflow    *workflow.Service
	Journal     *journal.Service
	Ticketbot   *ticketbot.Service
}

const defaultStoreTTL = int64(900)

func NewApp(ctx context.Context, migVersion int64, level *slog.LevelVar, logBuf *logging.BufferHandler) (*App, *logging.Persister, error) {
	cr := getCreds()
	tf := getTestFlags()
	if err := cr.validateBootstrap(); err != nil {
		return nil, nil, fmt.Errorf("validating credentials: %w", err)
	}

	ttl := defaultStoreTTL
	if tf.StoreTTLSeconds != 0 {
		ttl = tf.StoreTTLSeconds
	}
	slog.Info("using TTL", "ttl", ttl)

	s, err := CreateStores(ctx, cr, migVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing stores: %w", err)
	}
	r := s.Repos

	cfg, err := getStartupConfig(ctx, r.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("getting initial config: %w", err)
	}

	// Credentials come from the config (admin panel) with environment variables
	// taking precedence; env values are written back so the DB keeps a working copy.
	if locked := mergeEnvConfig(cfg, cr); len(locked) > 0 {
		slog.Info("config sourced from environment", "fields", locked)
		if cfg, err = r.Config.Upsert(ctx, cfg); err != nil {
			return nil, nil, fmt.Errorf("persisting env config: %w", err)
		}
	}
	if err := validateCreds(cfg, tf); err != nil {
		return nil, nil, fmt.Errorf("validating credentials: %w", err)
	}

	cw, err := psa.NewClient(ctx, psa.Config{
		PublicKey:  cfg.CwPublicKey,
		PrivateKey: cfg.CwPrivateKey,
		ClientID:   cfg.CwClientID,
		CompanyID:  cfg.CwCompanyID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating connectwise client: %w", err)
	}

	ms, err := makeMessageSender(ctx, tf.MockWebex, cfg.WebexSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("creating message sender: %w", err)
	}

	cws := cwsvc.New(s.Pool, r.CW, cw, ttl)
	ws := webexsvc.New(s.Pool, r.WebexRecipients, ms)

	nr := notifier.SvcParams{
		Cfg:           cfg,
		WebexSvc:      ws,
		NotifierRules: r.NotifierRules,
		Notifications: r.TicketNotifications,
		Forwards:      r.NotifierForwards,
		Pool:          s.Pool,
		MessageSender: ms,
		CWCompanyID:   cfg.CwCompanyID,
	}

	ns := notifier.New(nr)

	wfs := workflow.New(workflow.SvcParams{
		Cfg:         cfg,
		CWClient:    cw,
		Workflows:   r.Workflows,
		Runs:        r.WorkflowRuns,
		Webex:       ms,
		Recips:      r.WebexRecipients,
		CWCompanyID: cfg.CwCompanyID,
	})

	js := journal.New(r.TicketJournals, cfg)

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
			Auth:        authsvc.New(r.APIUser, r.Sessions, r.TOTPPending, r.TOTPRecovery, cfg),
			Config:      config.New(r.Config, cfg, level, logBuf),
			User:        user.New(r.APIUser, r.APIKey),
			Hooks:       webhooks.New(cw, cfg.RootURL),
			CW:          cws,
			Webex:       ws,
			Sync:        syncsvc.New(s.Pool, cws, ws, ns),
			Notifier:    notifier.New(nr),
			Workflow:    wfs,
			Journal:     js,
			Ticketbot:   ticketbot.New(cfg, cws, ns, wfs, js),
		},
	}, persister, nil
}
