package webex

import (
	"fmt"
)

func NewMessageToPerson(email, text string) Message {
	return Message{ToPersonEmail: email, Markdown: text}
}

func NewMessageToRoom(roomId, text string) Message {
	return Message{RoomId: roomId, Markdown: text}
}

func (c *Client) GetMessage(messageID string, params map[string]string) (*Message, error) {
	return GetOne[Message](c, fmt.Sprintf("messages/%s", messageID), params)
}

func (c *Client) PostMessage(message *Message) (*Message, error) {
	return Post[Message](c, "messages", message)
}
