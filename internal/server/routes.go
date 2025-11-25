package server

import (
	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/handler"
	"github.com/thecoretg/ticketbot/internal/middleware"
)

func (a *App) addRoutes(g *gin.Engine) {
	errh := middleware.ErrorHandler()
	auth := middleware.APIKeyAuth(a.Svc.User.Keys)
	cws := middleware.RequireConnectwiseSignature()

	sh := handler.NewSyncHandler(a.Svc.CW, a.Svc.Webex)
	g.POST("sync", sh.HandleSync)

	u := g.Group("users", errh, auth)
	uh := handler.NewUserHandler(a.Svc.User)
	registerUserRoutes(u, uh)

	c := g.Group("config", errh, auth)
	ch := handler.NewConfigHandler(a.Svc.Config)
	registerConfigRoutes(c, ch)

	cw := g.Group("cw", errh, auth)
	cwh := handler.NewCWHandler(a.Svc.CW)
	registerCWRoutes(cw, cwh)

	wx := g.Group("webex", errh, auth)
	wh := handler.NewWebexHandler(a.Svc.Webex)
	registerWebexRoutes(wx, wh)

	n := g.Group("notifiers", errh, auth)
	nh := handler.NewNotifierHandler(a.Stores.Notifiers, a.Stores.CW.Board, a.Stores.WebexRoom)
	registerNotifierRoutes(n, nh)

	tb := handler.NewTicketbotHandler(a.Svc.Ticketbot)
	g.POST("hooks/cw/tickets", tb.ProcessTicket, errh, cws)
}

func registerUserRoutes(r *gin.RouterGroup, h *handler.UserHandler) {
	r.GET("", h.ListUsers)
	r.GET(":id", h.GetUser)
	r.DELETE(":id")

	k := r.Group("keys")
	k.GET("", h.ListAPIKeys)
	k.GET(":id", h.GetAPIKey)
	k.POST("", h.AddAPIKey)
	k.DELETE(":id", h.DeleteAPIKey)
}

func registerConfigRoutes(r *gin.RouterGroup, h *handler.ConfigHandler) {
	r.GET("", h.Get)
	r.PUT("", h.Update)
}

func registerCWRoutes(r *gin.RouterGroup, h *handler.CWHandler) {
	b := r.Group("boards")
	b.GET("", h.ListBoards)
	b.GET(":id", h.GetBoard)
}

func registerWebexRoutes(r *gin.RouterGroup, h *handler.WebexHandler) {
	ro := r.Group("rooms")
	ro.GET("", h.ListRooms)
	ro.GET(":id", h.GetRoom)
}

func registerNotifierRoutes(r *gin.RouterGroup, h *handler.NotifierHandler) {
	r.GET("", h.ListNotifiers)
	r.POST("", h.AddNotifier)
}
