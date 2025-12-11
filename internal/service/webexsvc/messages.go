package webexsvc

import (
	"context"
	"fmt"

	"github.com/thecoretg/ticketbot/pkg/webex"
)

func (s *Service) GetMessage(ctx context.Context, msgID string) (*webex.Message, error) {
	msg, err := s.WebexClient.GetMessage(msgID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting full message details from webex: %w", err)
	}

	return msg, nil
}
