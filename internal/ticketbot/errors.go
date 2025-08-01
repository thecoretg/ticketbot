package ticketbot

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type ErrWasDeleted struct {
	ItemType string
	ItemID   int
}

func (e ErrWasDeleted) Error() string {
	return fmt.Sprintf("%s %d was deleted by external factors", e.ItemType, e.ItemID)
}

func ErrorHandler(exitOnError bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			c.JSON(http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": err.Error(),
			})
			c.Abort()
			slog.Error("error occurred in request", "error", err, "exitOnError", exitOnError)
			c.Writer.Flush()
			if exitOnError {
				os.Exit(1)
			}
		}
	}
}

func errorOutput(msg string) map[string]string {
	return map[string]string{
		"error": msg,
	}
}
