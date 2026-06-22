package webexsvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/thecoretg/tctg-go/webex"
)

var ErrMessageFromBot = errors.New("message is from the bot")

func (s *Service) GetMessage(ctx context.Context, payload *webex.MessageHookPayload) (*webex.Message, error) {
	data := payload.Data
	if data.PersonEmail == s.BotEmail {
		return nil, ErrMessageFromBot
	}

	msg, err := s.WebexClient.GetMessage(ctx, data.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting full message details from webex: %w", err)
	}

	return msg, nil
}

func (s *Service) GetAttachmentAction(ctx context.Context, payload *webex.MessageHookPayload) (*webex.AttachmentAction, error) {
	data := payload.Data
	if data.PersonEmail == s.BotEmail {
		return nil, ErrMessageFromBot
	}

	action, err := s.WebexClient.GetAttachmentAction(ctx, data.ID)
	if err != nil {
		return nil, fmt.Errorf("getting attachment action from webex: %w", err)
	}

	return action, nil
}

func (s *Service) PostMessage(ctx context.Context, msg *webex.Message) (*webex.Message, error) {
	return s.WebexClient.PostMessage(ctx, msg)
}
