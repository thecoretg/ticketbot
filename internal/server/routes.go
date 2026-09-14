package server

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/handlers"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/web"
)

func AddRoutes(a *App, g *gin.Engine, shutdown func()) {
	panelFS, _ := fs.Sub(web.StaticFiles, "static")
	g.StaticFS("/panel", http.FS(panelFS))

	auth := middleware.CombinedAuth(a.Stores.APIKey, a.Svc.Auth)

	g.GET("healthcheck", handlers.HandleHealthCheck) // authless ping for lightsail health checks
	g.GET("authtest", auth, handlers.HandleHealthCheck)

	ah := handlers.NewAuthHandler(a.Svc.Auth)
	g.POST("auth/login", ah.HandleLogin)
	g.POST("auth/logout", ah.HandleLogout)
	g.PUT("auth/password", auth, ah.HandleChangePassword)

	th := handlers.NewTOTPHandler(a.Svc.Auth)
	g.POST("auth/totp/verify", th.HandleVerify)
	g.GET("auth/totp", auth, th.HandleStatus)
	g.POST("auth/totp/setup", auth, th.HandleBeginSetup)
	g.PUT("auth/totp/setup", auth, th.HandleConfirmSetup)
	g.DELETE("auth/totp", auth, th.HandleDisable)

	s := g.Group("sync", auth)
	sh := handlers.NewSyncHandler(a.Svc.Sync, a.Config)
	registerSyncRoutes(s, sh)

	u := g.Group("users", auth)
	uh := handlers.NewUserHandler(a.Svc.User)
	registerUserRoutes(u, uh)

	c := g.Group("config", auth)
	ch := handlers.NewConfigHandler(a.Svc.Config)
	registerConfigRoutes(c, ch)

	cw := g.Group("cw", auth)
	cwh := handlers.NewCWHandler(a.Svc.CW)
	registerCWRoutes(cw, cwh)

	wx := g.Group("webex", auth)
	wh := handlers.NewWebexHandler(a.Svc.Webex)
	registerWebexRoutes(wx, wh)

	n := g.Group("notifiers", auth)
	nh := handlers.NewNotifierHandler(a.Svc.Notifier)
	registerNotifierRoutes(n, nh)

	wf := g.Group("workflows", auth)
	wfh := handlers.NewWorkflowHandler(a.Svc.Workflow, a.Svc.CW)
	registerWorkflowRoutes(wf, wfh)

	t := g.Group("tickets", auth)
	tkh := handlers.NewTicketsHandler(a.Svc.CW)
	registerTicketRoutes(t, tkh)

	lh := handlers.NewLogsHandler(a.LogBuffer)
	g.GET("logs", auth, lh.HandleList)

	adminh := handlers.NewAdminHandler(shutdown)
	g.POST("admin/restart", auth, adminh.HandleRestart)

	tb := handlers.NewTicketbotHandler(a.Svc.Ticketbot)
	hh := g.Group("hooks")
	registerHookRoutes(hh, tb)
}

func registerSyncRoutes(r *gin.RouterGroup, h *handlers.SyncHandler) {
	r.POST("", h.HandleSync)
	r.GET("status", h.HandleSyncStatus)
}

func registerUserRoutes(r *gin.RouterGroup, h *handlers.UserHandler) {
	r.GET("", h.ListUsers)
	r.GET("me", h.GetCurrentUser)
	r.GET(":id", h.GetUser)
	r.POST("", h.CreateUser)
	r.DELETE(":id", h.DeleteUser)

	k := r.Group("keys")
	k.GET("", h.ListAPIKeys)
	k.GET(":id", h.GetAPIKey)
	k.POST("", h.AddAPIKey)
	k.DELETE(":id", h.DeleteAPIKey)
}

func registerConfigRoutes(r *gin.RouterGroup, h *handlers.ConfigHandler) {
	r.GET("", h.Get)
	r.PUT("", h.Update)
}

func registerCWRoutes(r *gin.RouterGroup, h *handlers.CWHandler) {
	b := r.Group("boards")
	b.GET("", h.ListBoards)
	b.GET(":id", h.GetBoard)
	b.GET(":id/statuses", h.ListBoardStatuses)

	m := r.Group("members")
	m.GET("", h.ListMembers)
}

func registerWebexRoutes(r *gin.RouterGroup, h *handlers.WebexHandler) {
	ro := r.Group("rooms")
	ro.GET("", h.ListRecipients)
	ro.GET(":id", h.GetRoom)
}

func registerNotifierRoutes(r *gin.RouterGroup, h *handlers.NotifierHandler) {
	fw := r.Group("forwards")
	fw.GET("", h.ListForwards)
	fw.GET(":id", h.GetForward)
	fw.POST("", h.AddUserForward)
	fw.PUT(":id", h.UpdateUserForward)
	fw.DELETE(":id", h.DeleteUserForward)
}

func registerWorkflowRoutes(r *gin.RouterGroup, h *handlers.WorkflowHandler) {
	r.GET("", h.List)
	r.POST("", h.Create)
	r.POST("validate-condition", h.ValidateCondition)
	r.POST("evaluate-condition", h.EvaluateCondition)
	r.GET("board/:id", h.GetByBoard)
	r.GET(":id", h.Get)
	r.PUT(":id", h.Replace)
	r.DELETE(":id", h.Delete)
}

func registerTicketRoutes(r *gin.RouterGroup, h *handlers.TicketsHandler) {
	r.GET("", h.List)
	r.GET(":id", h.Get)
	r.GET(":id/raw", h.Raw)
	r.GET(":id/events", h.Events)
}

func registerHookRoutes(r *gin.RouterGroup, tb *handlers.TicketbotHandler) {
	r.POST("cw/tickets", middleware.RequireConnectwiseSignature(), tb.ProcessTicket)
}
