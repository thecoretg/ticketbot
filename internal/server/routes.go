package server

import (
	"io/fs"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/handlers"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/service/oauth"
	"github.com/thecoretg/ticketbot/internal/service/sso"
	"github.com/thecoretg/ticketbot/internal/web"
	"github.com/thecoretg/ticketbot/models"
)

// router registers routes on a ServeMux, wrapping each in the given middleware.
type router struct {
	mux *http.ServeMux
}

func (rt *router) handle(pattern string, h http.HandlerFunc, mws ...middleware.Middleware) {
	rt.mux.Handle(pattern, middleware.Chain(h, mws...))
}

// noCache makes browsers and Cloudflare revalidate the dashboard's assets on every load. The files
// are embedded in the binary and change with every deploy but keep the same names, so a plain
// max-age would serve the previous build's CSS and JS for hours. Revalidation is cheap: the file
// server answers If-Modified-Since with a 304 until the binary changes.
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

// requireSSOEnabled hides the Microsoft sign-in endpoints while the admin has SSO switched off.
func (a *App) requireSSOEnabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Svc.SSO.Enabled() {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireMCPEnabled hides the MCP server's OAuth endpoints while the admin has it switched off.
func (a *App) requireMCPEnabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Svc.OAuth.Enabled() {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewHandler builds the application's HTTP routes. shutdown is invoked by the admin restart route.
func NewHandler(a *App, shutdown func()) http.Handler {
	rt := &router{mux: http.NewServeMux()}
	// Every protected route runs auth; editor and admin add a role floor on top. Viewers get
	// every GET plus the pure workflow-condition helpers, editors mutate automation, admins run
	// the instance.
	auth := middleware.CombinedAuth(a.Stores.APIKey, a.Stores.APIUser, a.Svc.Auth, a.ssoAuth(), a.ssoUser)
	editor := middleware.RequireRole(models.RoleEditor)
	admin := middleware.RequireRole(models.RoleAdmin)

	// The dashboard lives at the root. ServeMux prefers the more specific API patterns, so this
	// catch-all only sees paths no route claims. /panel/ stays as a redirect for old bookmarks.
	panelFS, _ := fs.Sub(web.StaticFiles, "static")
	rt.mux.Handle("GET /", noCache(http.FileServerFS(panelFS)))
	rt.mux.Handle("GET /panel/", http.RedirectHandler("/", http.StatusMovedPermanently))

	rt.handle("GET /healthcheck", handlers.HandleHealthCheck) // authless ping for load balancer / container health checks
	rt.handle("GET /authtest", handlers.HandleHealthCheck, auth)

	ah := handlers.NewAuthHandler(a.Svc.Auth, a.Svc.SSO)
	rt.handle("POST /auth/login", ah.HandleLogin)
	rt.handle("POST /auth/logout", ah.HandleLogout)
	rt.handle("PUT /auth/password", ah.HandleChangePassword, auth)

	// Microsoft sign-in. Start and Callback exist only while SSO is configured and enabled; the
	// admin API below always exists so the dashboard can show setup state.
	ssoh := handlers.NewSSOHandler(a.Svc.SSO)
	rt.handle("GET /auth/methods", ssoh.Methods)
	if a.SSOAuth != nil {
		rt.mux.Handle("GET /auth/sso/start", a.requireSSOEnabled(a.SSOAuth.Start()))
		rt.mux.Handle("GET "+sso.CallbackPath, a.requireSSOEnabled(a.SSOAuth.Callback()))
	}
	rt.handle("GET /sso", ssoh.Status, auth, admin)
	rt.handle("POST /sso/test", ssoh.Test, auth, admin)
	rt.handle("PUT /sso/mappings", ssoh.UpsertMapping, auth, admin)
	rt.handle("DELETE /sso/mappings/{id}", ssoh.DeleteMapping, auth, admin)

	th := handlers.NewTOTPHandler(a.Svc.Auth)
	rt.handle("POST /auth/totp/verify", th.HandleVerify)
	rt.handle("GET /auth/totp", th.HandleStatus, auth)
	rt.handle("POST /auth/totp/setup", th.HandleBeginSetup, auth)
	rt.handle("PUT /auth/totp/setup", th.HandleConfirmSetup, auth)
	rt.handle("DELETE /auth/totp", th.HandleDisable, auth)

	// OAuth in front of the MCP endpoint (item 14). The consent screen is the dashboard at
	// /oauth/consent, so that path serves index.html; the view reads the forwarded query.
	mcp := a.requireMCPEnabled
	oh := handlers.NewOAuthHandler(a.Svc.OAuth)
	rt.handle("GET /.well-known/oauth-protected-resource", oh.ProtectedResourceMetadata, mcp)
	rt.handle("GET /.well-known/oauth-protected-resource"+oauth.MCPPath, oh.ProtectedResourceMetadata, mcp)
	rt.handle("GET /.well-known/oauth-authorization-server", oh.ServerMetadata, mcp)
	rt.handle("POST /oauth/register", oh.Register, mcp)
	rt.handle("GET /oauth/authorize", oh.Authorize, mcp)
	rt.handle("GET /oauth/authorize/info", oh.ConsentInfo, mcp, auth)
	rt.handle("POST /oauth/authorize/decide", oh.Decide, mcp, auth)
	rt.handle("POST /oauth/token", oh.Token, mcp)
	rt.handle("POST /oauth/revoke", oh.Revoke, mcp)
	rt.mux.Handle("GET "+oauth.ConsentPath, mcp(noCache(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, panelFS, "index.html")
	}))))
	// Connected apps exist whether or not MCP is on, so grants can be revoked after it is off.
	rt.handle("GET /users/me/grants", oh.ListMyGrants, auth)
	rt.handle("DELETE /users/me/grants/{grant_id}", oh.RevokeMyGrant, auth)
	// Admin paths sit under /users/grants like /users/keys, because /users/{id}/grants would
	// conflict with /users/keys/{id} in ServeMux.
	rt.handle("GET /users/grants/{id}", oh.ListUserGrants, auth, admin)
	rt.handle("DELETE /users/grants/{id}/{grant_id}", oh.RevokeUserGrant, auth, admin)

	sh := handlers.NewSyncHandler(a.Svc.Sync, a.Config)
	rt.handle("POST /sync", sh.HandleSync, auth, admin)
	rt.handle("GET /sync/status", sh.HandleSyncStatus, auth, admin)

	uh := handlers.NewUserHandler(a.Svc.User)
	rt.handle("GET /users", uh.ListUsers, auth, admin)
	rt.handle("GET /users/me", uh.GetCurrentUser, auth)
	rt.handle("GET /users/{id}", uh.GetUser, auth, admin)
	rt.handle("POST /users", uh.CreateUser, auth, admin)
	rt.handle("DELETE /users/{id}", uh.DeleteUser, auth, admin)
	rt.handle("PUT /users/{id}/role", uh.SetRole, auth, admin)
	rt.handle("GET /users/keys", uh.ListAPIKeys, auth, admin)
	rt.handle("GET /users/keys/{id}", uh.GetAPIKey, auth, admin)
	rt.handle("POST /users/keys", uh.AddAPIKey, auth, admin)
	rt.handle("DELETE /users/keys/{id}", uh.DeleteAPIKey, auth, admin)

	ch := handlers.NewConfigHandler(a.Svc.Config)
	// Every signed-in user reads config: the shell needs require_totp and master_dry_run.
	rt.handle("GET /config", ch.Get, auth)
	rt.handle("PUT /config", ch.Update, auth, admin)

	cwh := handlers.NewCWHandler(a.Svc.CW)
	rt.handle("GET /cw/boards", cwh.ListBoards, auth)
	rt.handle("GET /cw/boards/{id}", cwh.GetBoard, auth)
	rt.handle("GET /cw/boards/{id}/statuses", cwh.ListBoardStatuses, auth)
	rt.handle("GET /cw/members", cwh.ListMembers, auth)
	rt.handle("GET /cw/priorities", cwh.ListPriorities, auth)
	rt.handle("GET /cw/companies", cwh.ListCompanies, auth)
	rt.handle("GET /cw/contacts", cwh.ListContacts, auth)

	wh := handlers.NewWebexHandler(a.Svc.Webex)
	rt.handle("GET /webex/rooms", wh.ListRecipients, auth)
	rt.handle("GET /webex/rooms/{id}", wh.GetRoom, auth)

	nh := handlers.NewNotifierHandler(a.Svc.Notifier)
	rt.handle("GET /notifiers/forwards", nh.ListForwards, auth)
	rt.handle("GET /notifiers/forwards/{id}", nh.GetForward, auth)
	rt.handle("POST /notifiers/forwards", nh.AddUserForward, auth, editor)
	rt.handle("PUT /notifiers/forwards/{id}", nh.UpdateUserForward, auth, editor)
	rt.handle("DELETE /notifiers/forwards/{id}", nh.DeleteUserForward, auth, editor)

	wfh := handlers.NewWorkflowHandler(a.Svc.Workflow, a.Svc.CW, a.Svc.Notifier, a.Svc.Lists)
	rt.handle("GET /workflows", wfh.List, auth)
	rt.handle("POST /workflows", wfh.Create, auth, editor)
	rh := handlers.NewRunsHandler(a.Stores.WorkflowRuns, a.Stores.TicketEvents, a.Stores.CW.Board, a.Svc.Workflow)
	rt.handle("GET /workflows/runs", rh.List, auth)
	rt.handle("GET /workflows/runs/{run_id}", rh.Get, auth)
	trh := handlers.NewTransferHandler(a.Svc.Transfer)
	rt.handle("GET /workflows/export", trh.Export, auth)
	rt.handle("POST /workflows/import", trh.Import, auth, editor)
	rt.handle("GET /workflows/fields", wfh.Fields, auth)
	rt.handle("GET /workflows/placeholders", wfh.Placeholders, auth)
	rt.handle("POST /workflows/preview-message", wfh.PreviewMessage, auth)
	rt.handle("POST /workflows/validate-condition", wfh.ValidateCondition, auth)
	rt.handle("POST /workflows/parse-condition", wfh.ParseCondition, auth)
	rt.handle("POST /workflows/evaluate-condition", wfh.EvaluateCondition, auth)
	rt.handle("GET /workflows/board/{id}", wfh.GetByBoard, auth)
	rt.handle("GET /workflows/{id}", wfh.Get, auth)
	rt.handle("POST /workflows/{id}/simulate", wfh.Simulate, auth)
	rt.handle("PUT /workflows/{id}", wfh.Replace, auth, editor)
	rt.handle("DELETE /workflows/{id}", wfh.Delete, auth, editor)

	lsh := handlers.NewListsHandler(a.Svc.Lists)
	rt.handle("GET /lists", lsh.List, auth)
	rt.handle("POST /lists", lsh.Create, auth, editor)
	rt.handle("GET /lists/types", lsh.Types, auth)
	rt.handle("GET /lists/{id}", lsh.Get, auth)
	rt.handle("PUT /lists/{id}", lsh.Update, auth, editor)
	rt.handle("DELETE /lists/{id}", lsh.Delete, auth, editor)
	rt.handle("POST /lists/{id}/items", lsh.AddItem, auth, editor)
	rt.handle("DELETE /lists/{id}/items/{item_id}", lsh.RemoveItem, auth, editor)

	tkh := handlers.NewTicketsHandler(a.Svc.CW)
	rt.handle("GET /tickets", tkh.List, auth)
	rt.handle("GET /tickets/{id}", tkh.Get, auth)
	rt.handle("GET /tickets/{id}/raw", tkh.Raw, auth)
	rt.handle("GET /tickets/{id}/events", tkh.Events, auth)

	lh := handlers.NewLogsHandler(a.LogBuffer)
	rt.handle("GET /logs", lh.HandleList, auth, admin)

	adminh := handlers.NewAdminHandler(shutdown)
	rt.handle("POST /admin/restart", adminh.HandleRestart, auth, admin)

	ih := handlers.NewIntakeHandler(a.Svc.Intake)
	rt.handle("GET /intake", ih.List, auth, admin)
	rt.handle("GET /intake/stats", ih.Stats, auth, admin)
	rt.handle("GET /intake/hourly", ih.Hourly, auth, admin)
	rt.handle("POST /intake/{id}/retry", ih.Retry, auth, admin)
	rt.handle("POST /intake/{id}/discard", ih.Discard, auth, admin)

	tb := handlers.NewTicketbotHandler(a.Svc.Intake)
	rt.handle("POST /hooks/cw/tickets", tb.ProcessTicket, middleware.RequireConnectwiseSignature())

	return rt.mux
}
