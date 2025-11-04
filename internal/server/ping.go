package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type pingResponse struct {
	Result string `json:"result"`
}

func (cl *Client) ping(c *gin.Context) {
	res := pingResponse{Result: "success"}
	c.JSON(http.StatusOK, res)
}
