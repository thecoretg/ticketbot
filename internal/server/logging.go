package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type setDebugPayload struct {
	Enabled bool `json:"enabled"`
}

func (cl *Client) handleSetDebug(c *gin.Context) {
	p := &setDebugPayload{}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := cl.setDebug(c.Request.Context(), p.Enabled); err != nil {
		c.Error(fmt.Errorf("setting debug state: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

// setLogger sets the default logger and is run at server startup
func setLogger(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	j := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(j))
}

// setLogLevel is used for changing the log level outside of server startup
func setLogLevel(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	slog.SetLogLoggerLevel(level)
}
