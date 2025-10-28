package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/psa"

	"github.com/gin-gonic/gin"
)

type ErrWasDeleted struct {
	ItemType string
	ItemID   int
}

func (e ErrWasDeleted) Error() string {
	return fmt.Sprintf("%s %d was deleted by external factors", e.ItemType, e.ItemID)
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			if errors.Is(err, psa.ErrNotFound) {
				slog.Debug("ErrorHandler: connectwise 404 status received, not returning error")
				c.Status(http.StatusNoContent)
				return
			}
			slog.Error("error occurred in request", "error", err)
			c.Status(http.StatusInternalServerError)
			c.Abort()
			c.Writer.Flush()
		}
	}
}

func errorOutput(msg string) map[string]string {
	return map[string]string{
		"error": msg,
	}
}
