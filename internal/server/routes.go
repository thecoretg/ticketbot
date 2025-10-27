package server

func (s *Server) addRoutes() {
	s.GinEngine.GET("/", s.ping)
	s.GinEngine.POST("/preload", s.handlePreload, ErrorHandler(s.Config.ExitOnError), s.apiKeyAuth())

	settings := s.GinEngine.Group("/settings", ErrorHandler(s.Config.ExitOnError), s.apiKeyAuth())
	settings.POST("/attempt_notify", s.handleSetAttemptNotify)

	keys := s.GinEngine.Group("/keys", ErrorHandler(s.Config.ExitOnError), s.apiKeyAuth())
	keys.POST("/", s.handleCreateAPIKey)

	boards := s.GinEngine.Group("/boards", ErrorHandler(s.Config.ExitOnError), s.apiKeyAuth())
	boards.GET("/:board_id", s.handleGetBoard)
	boards.GET("/", s.handleListBoards)
	boards.PUT("/:board_id", s.handlePutBoard)
	boards.DELETE("/:board_id", s.handleDeleteBoard)

	rooms := s.GinEngine.Group("/rooms", ErrorHandler(s.Config.ExitOnError), s.apiKeyAuth())
	rooms.GET("/", s.handleListWebexRooms)

	hooks := s.GinEngine.Group("/hooks")
	cwHooks := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(s.Config.ExitOnError))
	cwHooks.POST("/tickets", s.handleTickets)
}
