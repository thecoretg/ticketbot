package handlers

import (
	"github.com/gin-gonic/gin"
)

func HandleHealthCheck(c *gin.Context) {
	res := struct {
		Result string `json:"result"`
	}{Result: "success"}

	outputJSON(c, res)
}
