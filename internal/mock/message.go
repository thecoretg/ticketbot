package mock

import (
	"github.com/thecoretg/ticketbot/pkg/webex"
)

// This isn't ideal at all, but I need it short term just to get this up and running.
// When I get time I will make a mock setup that doesn't actually call the Webex API.

type WebexClient struct {
	// real webex client used for listing rooms, but not posting messages
	webexClient *webex.Client
}

func NewWebexClient(token string) *WebexClient {
	return &WebexClient{
		webexClient: webex.NewClient(token),
	}
}

func (w *WebexClient) GetMessage(id string, params map[string]string) (*webex.Message, error) {
	return &webex.Message{}, nil
}

func (w *WebexClient) GetAttachmentAction(messageID string) (*webex.AttachmentAction, error) {
	return &webex.AttachmentAction{}, nil
}

func (w *WebexClient) PostMessage(message *webex.Message) (*webex.Message, error) {
	return message, nil
}

func (w *WebexClient) ListRooms(params map[string]string) ([]webex.Room, error) {
	return w.webexClient.ListRooms(params)
}

func (w *WebexClient) ListPeople(email string) ([]webex.Person, error) {
	return w.webexClient.ListPeople(email)
}
