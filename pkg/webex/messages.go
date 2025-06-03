package webex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

func NewMessageToPerson(email, text string) MessagePostBody {
	return MessagePostBody{Person: email, Markdown: text}
}

func NewMessageToRoom(roomId, text string) MessagePostBody {
	return MessagePostBody{RoomId: roomId, Markdown: text}
}

func (c *Client) GetMessage(ctx context.Context, messageId string) (*MessageGetResponse, error) {
	m := &MessageGetResponse{}
	if err := c.request(ctx, "GET", fmt.Sprintf("messages/%s", messageId), nil, m); err != nil {
		return nil, fmt.Errorf("getting message %s: %w", messageId, err)
	}

	return m, nil
}

func (c *Client) SendMessage(ctx context.Context, message MessagePostBody) error {
	j, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshaling message to json: %w", err)
	}

	p := bytes.NewReader(j)

	if err := c.request(ctx, "POST", "messages", p, nil); err != nil {
		return err
	}

	return nil
}
