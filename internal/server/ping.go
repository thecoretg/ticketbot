package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type pingResponse struct {
	Result string `json:"result"`
}

func (s *Server) addPingGroup() {
	ping := s.GinEngine.Group("/ping", ErrorHandler(s.Config.General.ExitOnError), s.APIKeyAuth())
	ping.GET("/", s.ping)
}

func (s *Server) ping(c *gin.Context) {
	res := pingResponse{Result: "success"}
	c.JSON(http.StatusOK, res)
}
