package mock

import (
	"context"

	"github.com/thecoretg/tctg-go/webex"
)

// This isn't ideal at all, but I need it short term just to get this up and running.
// When I get time I will make a mock setup that doesn't actually call the Webex API.

type WebexClient struct {
	// real webex client used for listing rooms, but not posting messages
	webexClient *webex.Client
}

func NewWebexClient(ctx context.Context, token string) (*WebexClient, error) {
	c, err := webex.NewClient(ctx, webex.Config{Token: token})
	if err != nil {
		return nil, err
	}

	return &WebexClient{
		webexClient: c,
	}, nil
}

func (w *WebexClient) GetMessage(ctx context.Context, id string, params map[string]string) (*webex.Message, error) {
	return &webex.Message{}, nil
}

func (w *WebexClient) GetAttachmentAction(ctx context.Context, messageID string) (*webex.AttachmentAction, error) {
	return &webex.AttachmentAction{}, nil
}

func (w *WebexClient) PostMessage(ctx context.Context, message *webex.Message) (*webex.Message, error) {
	return message, nil
}

func (w *WebexClient) ListRooms(ctx context.Context, params map[string]string) ([]webex.Room, error) {
	return w.webexClient.ListRooms(ctx, params)
}

func (w *WebexClient) ListPeople(ctx context.Context, email string) ([]webex.Person, error) {
	return w.webexClient.ListPeople(ctx, email)
}
