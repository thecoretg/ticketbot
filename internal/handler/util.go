package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func convertID(c *gin.Context) (int, error) {
	s := c.Param("id")
	return strconv.Atoi(s)
}
