package handler

import (
	"github.com/gin-gonic/gin"
)

func HandlePing(c *gin.Context) {
	res := struct {
		Result string `json:"result"`
	}{Result: "success"}

	outputJSON(c, res)
}
