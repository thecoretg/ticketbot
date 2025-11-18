package newserver

import (
	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/handler"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/service/user"
)

func (a *App) addRoutes(r *gin.Engine) {
	errh := middleware.ErrorHandler()
	auth := middleware.APIKeyAuth(a.Stores.APIKey)

	th := handler.NewTicketHandler(a.Svc.Ticket)
	r.POST("hooks/cw/tickets", th.ProcessTicket, errh, middleware.RequireConnectwiseSignature())

	ch := handler.NewConfigHandler(a.Svc.Config)
	cfg := r.Group("config", errh, auth)
	cfg.GET("", ch.Get)
	cfg.PUT("", ch.Update)
}

func registerUserRoutes(g *gin.Engine, svc *user.Service) {
	h := handler.NewUserHandler(svc)
	u := g.Group("users", middleware.ErrorHandler(), middleware.APIKeyAuth(svc.Keys))
	u.GET("", h.ListUsers)
	u.GET(":id", h.GetUser)
	u.DELETE(":id")

	k := u.Group("keys")
	k.GET("", h.ListAPIKeys)
	k.GET(":id", h.GetAPIKey)
	k.POST("", h.AddAPIKey)
	k.DELETE(":id", h.DeleteAPIKey)
}
