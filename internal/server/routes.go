package server

func (cl *Client) addRoutes() {
	eh := ErrorHandler()
	au := cl.apiKeyAuth()

	// Health Check
	cl.Server.GET("", cl.ping)

	// State
	cl.Server.GET("state", cl.handleGetState, eh, au)

	// Config
	c := cl.Server.Group("config", eh, au)
	c.GET("", cl.handleGetConfig)
	c.PUT("", cl.handlePutConfig)

	// Sync
	s := cl.Server.Group("sync", eh, au)
	s.POST("tickets", cl.handleSyncTickets)
	s.POST("webex_rooms", cl.handleSyncWebexRooms)

	// API Keys
	cl.Server.POST("keys", cl.handleCreateAPIKey, eh, au)

	// Boards
	b := cl.Server.Group("boards", eh, au)
	b.GET("", cl.handleListBoards)
	b.GET(":board_id", cl.handleGetBoard)
	b.PUT(":board_id", cl.handlePutBoard)
	b.DELETE(":board_id", cl.handleDeleteBoard)

	// Webex Rooms
	cl.Server.GET("rooms", cl.handleListWebexRooms, eh, au)

	// Webhooks
	sig := requireValidCWSignature()
	h := cl.Server.Group("hooks", eh, sig)
	h.POST("cw/tickets", cl.handleTickets)
}
