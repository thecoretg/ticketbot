package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/tctg-go/entra"
	"github.com/thecoretg/ticketbot/internal/env"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/authsvc"
	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/lists"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/sso"
	"github.com/thecoretg/ticketbot/internal/service/syncsvc"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/internal/service/user"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/internal/service/webhooks"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type App struct {
	Env                     *env.Env
	Stores                  *repos.AllRepos
	CWClient                *psa.Client
	MessageSender           repos.MessageSender
	Pool                    *pgxpool.Pool
	Config                  *models.Config
	Svc                     *Services
	CurrentMigrationVersion int64
	LogBuffer               *logging.BufferHandler
	// SSOAuth is nil when the ENTRA_* environment variables are not set.
	SSOAuth *entra.Auth[*models.APIUser]
}

// ssoAuth hands the middleware the Entra auth, or a true nil when SSO is not configured. A typed
// nil pointer must not become a non-nil interface.
func (a *App) ssoAuth() middleware.SSOAuth {
	if a.SSOAuth == nil {
		return nil
	}
	return a.SSOAuth
}

// ssoUser reads the user entra.RequireAuth placed in the request context.
func (a *App) ssoUser(ctx context.Context) (*models.APIUser, bool) {
	return entra.UserFromContext[*models.APIUser](ctx)
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
	SSO       *sso.Service
}

func NewApp(ctx context.Context, e *env.Env, migVersion int64, level *slog.LevelVar, logBuf *logging.BufferHandler) (*App, *logging.Persister, error) {
	slog.Info("using store TTL", "ttl", e.StoreTTL)

	cw, err := psa.NewClient(ctx, e.CW)
	if err != nil {
		return nil, nil, fmt.Errorf("creating connectwise client: %w", err)
	}

	ms, err := makeMessageSender(ctx, e.MockWebex, e.WebexSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("creating message sender: %w", err)
	}

	s, err := InitPostgresStores(ctx, e, migVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing stores: %w", err)
	}
	r := s.Repos

	cfg, err := getStartupConfig(ctx, r.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("getting initial config: %w", err)
	}

	cws := cwsvc.New(s.Pool, r.CW, r.TicketEvents, cw, e.CW.CompanyID, e.StoreTTL)
	ws := webexsvc.New(s.Pool, r.WebexRecipients, ms)

	nr := notifier.SvcParams{
		Cfg:           cfg,
		WebexSvc:      ws,
		Notifications: r.TicketNotifications,
		Forwards:      r.NotifierForwards,
		Pool:          s.Pool,
		MessageSender: ms,
		CWCompanyID:   e.CW.CompanyID,
	}

	ns := notifier.New(nr)
	cfgSvc := config.New(r.Config, cfg, level, logBuf, e.Entra.Configured())
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

	ssoSvc, ssoAuth, err := makeSSO(ctx, e, r, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("configuring sso: %w", err)
	}

	return &App{
		Env:           e,
		Config:        cfg,
		Stores:        r,
		Pool:          s.Pool,
		CWClient:      cw,
		MessageSender: ms,
		LogBuffer:     logBuf,
		SSOAuth:       ssoAuth,
		Svc: &Services{
			Auth:      authsvc.New(r.APIUser, r.Sessions, r.TOTPPending, r.TOTPRecovery, cfg, e.InitialAdminEmail, e.Entra.Configured()),
			Config:    cfgSvc,
			User:      user.New(r.APIUser, r.APIKey, e.InitialAdminEmail),
			Hooks:     webhooks.New(cw, e.RootURL),
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
			SSO:   ssoSvc,
		},
	}, persister, nil
}

// makeSSO builds the sso service and, when the ENTRA_* variables are present, the entra.Auth
// around it. Discovery is lazy, so a Microsoft outage cannot stop the app from starting.
func makeSSO(ctx context.Context, e *env.Env, r *repos.AllRepos, cfg *models.Config) (*sso.Service, *entra.Auth[*models.APIUser], error) {
	svc, err := sso.New(ctx, sso.Params{Users: r.APIUser, Mappings: r.SSORoleMappings, Cfg: cfg, Entra: e.Entra, RootURL: e.RootURL, BreakGlassEmail: e.InitialAdminEmail})
	if err != nil {
		return nil, nil, err
	}
	if !e.Entra.Configured() {
		slog.Info("entra sso not configured; password sign-in only")
		return svc, nil, nil
	}

	auth, err := entra.New(ctx, entra.Config{
		TenantID:     e.Entra.TenantID,
		ClientID:     e.Entra.ClientID,
		ClientSecret: e.Entra.ClientSecret,
		BaseURL:      e.RootURL,
		CallbackPath: sso.CallbackPath,
		LoginPath:    sso.PanelPath,
		SuccessPath:  sso.PanelPath,
		Authorize:    svc.Authorize,
		Sessions:     r.SSO,
		States:       r.SSO,
		Logger:       slog.Default(),
		Unauthorized: middleware.WriteUnauthorized,
	}, svc)
	if err != nil {
		return nil, nil, err
	}
	svc.SetAuth(auth)
	slog.Info("entra sso configured", "redirect_uri", auth.RedirectURI(), "enabled", cfg.SSOEnabled)
	return svc, auth, nil
}
