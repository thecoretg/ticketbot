package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func errorOutput(err error) gin.H {
	return gin.H{"error": err}
}

func badIntErrorOutput(s string) gin.H {
	return errorOutput(fmt.Errorf("%s is not a valid integer", s))
}
