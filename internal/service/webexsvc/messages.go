package webexsvc

import (
	"context"

	"github.com/thecoretg/tctg-go/webex"
)

func (s *Service) PostMessage(ctx context.Context, msg *webex.Message) (*webex.Message, error) {
	return s.WebexClient.PostMessage(ctx, msg)
}
