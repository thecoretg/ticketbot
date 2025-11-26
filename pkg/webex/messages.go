package webex

import (
	"fmt"
)

func NewMessageToPerson(email, text string) Message {
	return Message{ToPersonEmail: email, Markdown: text, RecipientName: email, RecipientType: "person"}
}

func NewMessageToRoom(roomId, roomName, text string) Message {
	return Message{RoomId: roomId, Markdown: text, RecipientName: roomName, RecipientType: "room"}
}

func (c *Client) GetMessage(messageID string, params map[string]string) (*Message, error) {
	return GetOne[Message](c, fmt.Sprintf("messages/%s", messageID), params)
}

func (c *Client) PostMessage(message *Message) (*Message, error) {
	return Post[Message](c, "messages", message)
}
