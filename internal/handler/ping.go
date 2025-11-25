package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandlePing(c *gin.Context) {
	res := struct {
		Result string `json:"result"`
	}{Result: "success"}

	c.JSON(http.StatusOK, res)
}
