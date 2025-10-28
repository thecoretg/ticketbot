package server

func (cl *Client) addRoutes() {
	// health check endpoint
	cl.Server.GET("/", cl.ping)

	state := cl.Server.Group("/state", ErrorHandler(cl.Config.ExitOnError), cl.apiKeyAuth())
	state.GET("/", cl.handleGetState)
	state.POST("/debug", cl.handleSetDebug)

	sc := cl.Server.Group("/sync", ErrorHandler(cl.Config.ExitOnError), cl.apiKeyAuth())
	sc.POST("/tickets", cl.handleSyncTickets)
	sc.POST("webex_rooms", cl.handleSyncWebexRooms)

	settings := cl.Server.Group("/settings", ErrorHandler(cl.Config.ExitOnError), cl.apiKeyAuth())
	settings.POST("/attempt_notify", cl.handleSetAttemptNotify)

	keys := cl.Server.Group("/keys", ErrorHandler(cl.Config.ExitOnError), cl.apiKeyAuth())
	keys.POST("/", cl.handleCreateAPIKey)

	boards := cl.Server.Group("/boards", ErrorHandler(cl.Config.ExitOnError), cl.apiKeyAuth())
	boards.GET("/:board_id", cl.handleGetBoard)
	boards.GET("/", cl.handleListBoards)
	boards.PUT("/:board_id", cl.handlePutBoard)
	boards.DELETE("/:board_id", cl.handleDeleteBoard)

	rooms := cl.Server.Group("/rooms", ErrorHandler(cl.Config.ExitOnError), cl.apiKeyAuth())
	rooms.GET("/", cl.handleListWebexRooms)

	hooks := cl.Server.Group("/hooks")
	cwHooks := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(cl.Config.ExitOnError))
	cwHooks.POST("/tickets", cl.handleTickets)
}
