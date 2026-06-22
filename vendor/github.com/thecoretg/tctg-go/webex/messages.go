package webex

import (
	"context"
	"fmt"
)

func NewMessageToPerson(email, text string) Message {
	return Message{ToPersonEmail: email, Markdown: text, RecipientName: email, RecipientType: "person"}
}

func NewMessageToRoom(roomID, roomName, text string) Message {
	return Message{RoomID: roomID, Markdown: text, RecipientName: roomName, RecipientType: "room"}
}

func (c *Client) GetMessage(ctx context.Context, messageID string, params map[string]string) (*Message, error) {
	return get[Message](ctx, c, fmt.Sprintf("messages/%s", messageID), params)
}

func (c *Client) PostMessage(ctx context.Context, message *Message) (*Message, error) {
	return post[Message](ctx, c, "messages", message)
}

func (c *Client) GetAttachmentAction(ctx context.Context, messageID string) (*AttachmentAction, error) {
	return get[AttachmentAction](ctx, c, fmt.Sprintf("attachment/actions/%s", messageID), nil)
}
