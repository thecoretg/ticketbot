package ticketbot

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/thecoretg/ticketbot/connectwise"

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
			if errors.Is(err, connectwise.ErrNotFound) {
				slog.Debug("ErrorHandler: connectwise 404 status received, not returning error")
				c.Status(http.StatusNoContent)
				return
			}
			slog.Error("error occurred in request", "error", err, "exitOnError", exitOnError)
			c.Status(http.StatusInternalServerError)
			c.Abort()
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
