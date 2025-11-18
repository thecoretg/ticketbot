package newserver

import (
	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/handler"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/internal/service/user"
)

func (a *App) addRoutes(g *gin.Engine) {
	errh := middleware.ErrorHandler()
	auth := middleware.APIKeyAuth(a.Svc.User.Keys)
	cws := middleware.RequireConnectwiseSignature()

	u := g.Group("users", errh, auth)
	registerUserRoutes(u, a.Svc.User)

	c := g.Group("config", errh, auth)
	registerConfigRoutes(c, a.Svc.Config)

	b := g.Group("boards", errh, auth)
	registerBoardRoutes(b, a.Stores.CW.Board)

	th := handler.NewTicketHandler(a.Svc.Ticket)
	g.POST("hooks/cw/tickets", th.ProcessTicket, errh, cws)

	n := g.Group("notifiers", errh, auth)
	registerNotifierRoutes(n, a.Stores.Notifiers, a.Stores.CW.Board, a.Stores.WebexRoom)
}

func registerUserRoutes(r *gin.RouterGroup, svc *user.Service) {
	h := handler.NewUserHandler(svc)
	r.GET("", h.ListUsers)
	r.GET(":id", h.GetUser)
	r.DELETE(":id")

	k := r.Group("keys")
	k.GET("", h.ListAPIKeys)
	k.GET(":id", h.GetAPIKey)
	k.POST("", h.AddAPIKey)
	k.DELETE(":id", h.DeleteAPIKey)
}

func registerConfigRoutes(r *gin.RouterGroup, svc *config.Service) {
	h := handler.NewConfigHandler(svc)
	r.GET("", h.Get)
	r.PUT("", h.Update)
}

func registerBoardRoutes(r *gin.RouterGroup, rp models.BoardRepository) {
	h := handler.NewBoardHandler(rp)
	r.GET("", h.ListBoards)
	r.GET(":id", h.GetBoard)
}

func registerNotifierRoutes(r *gin.RouterGroup, nr models.NotifierRepository, br models.BoardRepository, wr models.WebexRoomRepository) {
	h := handler.NewNotifierHandler(nr, br, wr)
	r.GET("", h.ListNotifiers)
	r.POST("", h.AddNotifier)
}
