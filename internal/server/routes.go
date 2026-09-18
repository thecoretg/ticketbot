package server

import (
	"io/fs"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/handlers"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/web"
)

// router registers routes on a ServeMux, wrapping each in the given middleware.
type router struct {
	mux *http.ServeMux
}

func (rt *router) handle(pattern string, h http.HandlerFunc, mws ...middleware.Middleware) {
	rt.mux.Handle(pattern, middleware.Chain(h, mws...))
}

// NewHandler builds the application's HTTP routes. shutdown is invoked by the admin restart route.
func NewHandler(a *App, shutdown func()) http.Handler {
	rt := &router{mux: http.NewServeMux()}
	auth := middleware.CombinedAuth(a.Stores.APIKey, a.Svc.Auth)

	panelFS, _ := fs.Sub(web.StaticFiles, "static")
	rt.mux.Handle("GET /panel/", http.StripPrefix("/panel", http.FileServerFS(panelFS)))

	rt.handle("GET /healthcheck", handlers.HandleHealthCheck) // authless ping for load balancer / container health checks
	rt.handle("GET /authtest", handlers.HandleHealthCheck, auth)

	ah := handlers.NewAuthHandler(a.Svc.Auth)
	rt.handle("POST /auth/login", ah.HandleLogin)
	rt.handle("POST /auth/logout", ah.HandleLogout)
	rt.handle("PUT /auth/password", ah.HandleChangePassword, auth)

	th := handlers.NewTOTPHandler(a.Svc.Auth)
	rt.handle("POST /auth/totp/verify", th.HandleVerify)
	rt.handle("GET /auth/totp", th.HandleStatus, auth)
	rt.handle("POST /auth/totp/setup", th.HandleBeginSetup, auth)
	rt.handle("PUT /auth/totp/setup", th.HandleConfirmSetup, auth)
	rt.handle("DELETE /auth/totp", th.HandleDisable, auth)

	sh := handlers.NewSyncHandler(a.Svc.Sync, a.Config)
	rt.handle("POST /sync", sh.HandleSync, auth)
	rt.handle("GET /sync/status", sh.HandleSyncStatus, auth)

	uh := handlers.NewUserHandler(a.Svc.User)
	rt.handle("GET /users", uh.ListUsers, auth)
	rt.handle("GET /users/me", uh.GetCurrentUser, auth)
	rt.handle("GET /users/{id}", uh.GetUser, auth)
	rt.handle("POST /users", uh.CreateUser, auth)
	rt.handle("DELETE /users/{id}", uh.DeleteUser, auth)
	rt.handle("GET /users/keys", uh.ListAPIKeys, auth)
	rt.handle("GET /users/keys/{id}", uh.GetAPIKey, auth)
	rt.handle("POST /users/keys", uh.AddAPIKey, auth)
	rt.handle("DELETE /users/keys/{id}", uh.DeleteAPIKey, auth)

	ch := handlers.NewConfigHandler(a.Svc.Config)
	rt.handle("GET /config", ch.Get, auth)
	rt.handle("PUT /config", ch.Update, auth)

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
	rt.handle("POST /notifiers/forwards", nh.AddUserForward, auth)
	rt.handle("PUT /notifiers/forwards/{id}", nh.UpdateUserForward, auth)
	rt.handle("DELETE /notifiers/forwards/{id}", nh.DeleteUserForward, auth)

	wfh := handlers.NewWorkflowHandler(a.Svc.Workflow, a.Svc.CW, a.Svc.Notifier, a.Svc.Lists)
	rt.handle("GET /workflows", wfh.List, auth)
	rt.handle("POST /workflows", wfh.Create, auth)
	rt.handle("GET /workflows/fields", wfh.Fields, auth)
	rt.handle("GET /workflows/placeholders", wfh.Placeholders, auth)
	rt.handle("POST /workflows/validate-condition", wfh.ValidateCondition, auth)
	rt.handle("POST /workflows/parse-condition", wfh.ParseCondition, auth)
	rt.handle("POST /workflows/evaluate-condition", wfh.EvaluateCondition, auth)
	rt.handle("GET /workflows/board/{id}", wfh.GetByBoard, auth)
	rt.handle("GET /workflows/{id}", wfh.Get, auth)
	rt.handle("POST /workflows/{id}/simulate", wfh.Simulate, auth)
	rt.handle("PUT /workflows/{id}", wfh.Replace, auth)
	rt.handle("DELETE /workflows/{id}", wfh.Delete, auth)

	lsh := handlers.NewListsHandler(a.Svc.Lists)
	rt.handle("GET /lists", lsh.List, auth)
	rt.handle("POST /lists", lsh.Create, auth)
	rt.handle("GET /lists/types", lsh.Types, auth)
	rt.handle("GET /lists/{id}", lsh.Get, auth)
	rt.handle("PUT /lists/{id}", lsh.Update, auth)
	rt.handle("DELETE /lists/{id}", lsh.Delete, auth)
	rt.handle("POST /lists/{id}/items", lsh.AddItem, auth)
	rt.handle("DELETE /lists/{id}/items/{item_id}", lsh.RemoveItem, auth)

	tkh := handlers.NewTicketsHandler(a.Svc.CW)
	rt.handle("GET /tickets", tkh.List, auth)
	rt.handle("GET /tickets/{id}", tkh.Get, auth)
	rt.handle("GET /tickets/{id}/raw", tkh.Raw, auth)
	rt.handle("GET /tickets/{id}/events", tkh.Events, auth)

	lh := handlers.NewLogsHandler(a.LogBuffer)
	rt.handle("GET /logs", lh.HandleList, auth)

	adminh := handlers.NewAdminHandler(shutdown)
	rt.handle("POST /admin/restart", adminh.HandleRestart, auth)

	tb := handlers.NewTicketbotHandler(a.Svc.Ticketbot)
	rt.handle("POST /hooks/cw/tickets", tb.ProcessTicket, middleware.RequireConnectwiseSignature())

	return rt.mux
}
