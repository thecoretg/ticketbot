package middleware

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

func RequireConnectwiseSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Error(fmt.Errorf("reading request body: %w", err))
			c.Next()
			c.Abort()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		valid, err := psa.ValidateWebhook(c.Request)
		if err != nil || !valid {
			c.Error(fmt.Errorf("invalid ConnectWise webhook signature: %w", err))
			c.Next()
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body for further processing
		c.Next()
	}
}
