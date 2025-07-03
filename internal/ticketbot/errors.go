package ticketbot

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"os"
	"tctg-automation/pkg/connectwise"
)

type ErrWasDeleted struct {
	ItemType string
	ItemID   int
}

func (e ErrWasDeleted) Error() string {
	return fmt.Sprintf("%s %d was deleted by external factors", e.ItemType, e.ItemID)
}

// checks for specific errors to reduce repetitive connectwise error checking
func checkCWError(msg, itemType string, err error, itemID int) error {
	var notFoundErr *connectwise.ErrNotFound

	switch {
	case errors.As(err, &notFoundErr):
		return ErrWasDeleted{
			ItemType: itemType,
			ItemID:   itemID,
		}
	default:
		return fmt.Errorf("%s: %w", msg, err)
	}
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
