package oldserver

import "github.com/thecoretg/ticketbot/internal/middleware"

func (cl *Client) addRoutes() {
	auth := middleware.NoOp()
	if !cl.testing.skipAuth {
		// TODO: DONT SKIP: add the real auth handler
	}

	errh := middleware.ErrorHandler()

	// Health Check
	cl.Server.GET("", cl.ping) // Authless ping for Lightsail health checks
	cl.Server.GET("authtest", auth)

	// State
	cl.Server.GET("state", cl.handleGetState, errh, auth)

	// Config
	c := cl.Server.Group("config", errh, auth)
	c.GET("", cl.handleGetConfig)
	c.PUT("", cl.handlePutConfig)

	// Sync
	s := cl.Server.Group("sync", errh, auth)
	s.POST("tickets", cl.handleSyncTickets)
	s.POST("webex_rooms", cl.handleSyncWebexRooms)
	s.POST("boards", cl.handleSyncBoards)

	// API Keys
	cl.Server.POST("keys", cl.handleCreateAPIKey, errh, auth)

	// Boards
	b := cl.Server.Group("boards", errh, auth)
	b.GET("", cl.handleListBoards)
	b.GET(":board_id", cl.handleGetBoard)

	// Webex Rooms
	cl.Server.GET("rooms", cl.handleListWebexRooms, errh, auth)

	// Notifiers
	n := cl.Server.Group("notifiers", errh, auth)
	n.GET("", cl.handleListNotifiers)
	n.POST("", cl.handlePostNotifier)
	n.GET(":notifier_id", cl.handleGetNotifier)
	n.DELETE(":notifier_id", cl.handleDeleteNotifier)

	// Webhooks
	sig := middleware.RequireConnectwiseSignature()
	h := cl.Server.Group("hooks", errh, sig)
	h.POST("cw/tickets", cl.handleTickets)
}
