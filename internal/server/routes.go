package server

func (s *Server) addRoutes() {
	ping := s.GinEngine.Group("/ping", ErrorHandler(s.Config.General.ExitOnError), s.apiKeyAuth())
	ping.GET("/", s.ping)

	keys := s.GinEngine.Group("/keys", ErrorHandler(s.Config.General.ExitOnError), s.apiKeyAuth())
	keys.POST("/", s.handleCreateAPIKey)

	boards := s.GinEngine.Group("/boards", ErrorHandler(s.Config.General.ExitOnError), s.apiKeyAuth())
	boards.GET("/:board_id", s.handleGetBoard)
	boards.GET("/", s.handleListBoards)
	boards.PUT("/:board_id", s.handlePutBoard)
	boards.DELETE("/:board_id", s.handleDeleteBoard)

	rooms := s.GinEngine.Group("/rooms", ErrorHandler(s.Config.General.ExitOnError), s.apiKeyAuth())
	rooms.GET("/", s.listWebexRooms)

	hooks := s.GinEngine.Group("/hooks")
	cwHooks := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(s.Config.General.ExitOnError))
	cwHooks.POST("/tickets", s.handleTickets)
}
