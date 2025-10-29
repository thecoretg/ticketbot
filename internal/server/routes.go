package server

func (cl *Client) addRoutes() {
	// Health Check
	cl.Server.GET("", cl.ping)

	// State
	cl.Server.GET("state", cl.handleGetState, ErrorHandler(), cl.apiKeyAuth())

	// Config
	cl.Server.GET("config", cl.handleGetConfig, ErrorHandler(), cl.apiKeyAuth())
	cl.Server.PUT("config", cl.handlePutConfig, ErrorHandler(), cl.apiKeyAuth())

	// Sync
	cl.Server.POST("sync/tickets", cl.handleSyncTickets, ErrorHandler(), cl.apiKeyAuth())
	cl.Server.POST("sync/webex_rooms", cl.handleSyncWebexRooms, ErrorHandler(), cl.apiKeyAuth())

	// API Keys
	cl.Server.POST("keys", cl.handleCreateAPIKey, ErrorHandler(), cl.apiKeyAuth())

	// Boards
	cl.Server.GET("boards", cl.handleListBoards, ErrorHandler(), cl.apiKeyAuth())
	cl.Server.GET("boards/:board_id", cl.handleGetBoard, ErrorHandler(), cl.apiKeyAuth())
	cl.Server.PUT("boards/:board_id", cl.handlePutBoard, ErrorHandler(), cl.apiKeyAuth())
	cl.Server.DELETE("boards/:board_id", cl.handleDeleteBoard, ErrorHandler(), cl.apiKeyAuth())

	// Webex Rooms
	cl.Server.GET("rooms", cl.handleListWebexRooms)

	// ConnectWise Webhooks
	cl.Server.POST("hooks/cw/tickets", requireValidCWSignature(), ErrorHandler())
}
