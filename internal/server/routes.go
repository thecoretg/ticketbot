package server

func (cl *Client) addRoutes() {
	cl.Server.GET("/", cl.ping)
	cl.Server.POST("/preload", cl.handlePreload, ErrorHandler(cl.Config.ExitOnError), cl.apiKeyAuth())

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
