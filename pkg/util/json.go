package util

import "github.com/gin-gonic/gin"

func ErrorJSON(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
