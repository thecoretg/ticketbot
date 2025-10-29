package server

func (cl *Client) addRoutes() {
	// health check endpoint
	cl.Server.GET("/", cl.ping)

	state := cl.Server.Group("/state", ErrorHandler(), cl.apiKeyAuth())
	state.GET("/", cl.handleGetState)

	config := cl.Server.Group("/config", ErrorHandler(), cl.apiKeyAuth())
	config.GET("/", cl.handleGetConfig)
	config.PUT("/", cl.handlePutConfig)

	sc := cl.Server.Group("/sync", ErrorHandler(), cl.apiKeyAuth())
	sc.POST("/tickets", cl.handleSyncTickets)
	sc.POST("webex_rooms", cl.handleSyncWebexRooms)

	settings := cl.Server.Group("/settings", ErrorHandler(), cl.apiKeyAuth())
	settings.POST("/attempt_notify", cl.handleSetAttemptNotify)

	keys := cl.Server.Group("/keys", ErrorHandler(), cl.apiKeyAuth())
	keys.POST("/", cl.handleCreateAPIKey)

	boards := cl.Server.Group("/boards", ErrorHandler(), cl.apiKeyAuth())
	boards.GET("/:board_id", cl.handleGetBoard)
	boards.GET("/", cl.handleListBoards)
	boards.PUT("/:board_id", cl.handlePutBoard)
	boards.DELETE("/:board_id", cl.handleDeleteBoard)

	rooms := cl.Server.Group("/rooms", ErrorHandler(), cl.apiKeyAuth())
	rooms.GET("/", cl.handleListWebexRooms)

	hooks := cl.Server.Group("/hooks")
	cwHooks := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler())
	cwHooks.POST("/tickets", cl.handleTickets)
}
