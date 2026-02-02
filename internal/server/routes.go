package server

import (
	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/handlers"
	"github.com/thecoretg/ticketbot/internal/middleware"
)

func AddRoutes(a *App, g *gin.Engine) {
	auth := middleware.APIKeyAuth(a.Svc.User.Keys)

	g.GET("healthcheck", handlers.HandleHealthCheck) // authless ping for lightsail health checks
	g.GET("authtest", auth, handlers.HandleHealthCheck)

	s := g.Group("sync", auth)
	sh := handlers.NewSyncHandler(a.Svc.Sync)
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

	m := r.Group("members")
	m.GET("", h.ListMembers)
}

func registerWebexRoutes(r *gin.RouterGroup, h *handlers.WebexHandler) {
	ro := r.Group("rooms")
	ro.GET("", h.ListRecipients)
	ro.GET(":id", h.GetRoom)
}

func registerNotifierRoutes(r *gin.RouterGroup, h *handlers.NotifierHandler) {
	ru := r.Group("rules")
	ru.GET("", h.ListNotifierRules)
	ru.GET(":id", h.GetNotifierRule)
	ru.POST("", h.AddNotifierRule)
	ru.DELETE(":id", h.DeleteNotifierRule)

	fw := r.Group("forwards")
	fw.GET("", h.ListForwards)
	fw.GET(":id", h.GetForward)
	fw.POST("", h.AddUserForward)
	fw.DELETE(":id", h.DeleteUserForward)
}

func registerHookRoutes(r *gin.RouterGroup, tb *handlers.TicketbotHandler) {
	r.POST("cw/tickets", middleware.RequireConnectwiseSignature(), tb.ProcessTicket)
}
