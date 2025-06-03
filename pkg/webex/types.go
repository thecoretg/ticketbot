package webex

import "time"

type Webhook struct {
	Name      string `json:"name"`
	TargetUrl string `json:"targetUrl"`
	Resource  string `json:"resource"`
	Event     string `json:"event"`
	Filter    string `json:"filter"`
}

type WebhooksGetResponse struct {
	Items []Webhook `json:"items"`
}

type MessageWebhookBody struct {
	Id        string    `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	TargetUrl string    `json:"targetUrl,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Event     string    `json:"event,omitempty"`
	Filter    string    `json:"filter,omitempty"`
	OrgId     string    `json:"orgId,omitempty"`
	CreatedBy string    `json:"createdBy,omitempty"`
	AppId     string    `json:"appId,omitempty"`
	OwnedBy   string    `json:"ownedBy,omitempty"`
	Status    string    `json:"status,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	ActorId   string    `json:"actorId,omitempty"`
	Data      struct {
		Id          string    `json:"id,omitempty"`
		RoomId      string    `json:"roomId,omitempty"`
		RoomType    string    `json:"roomType,omitempty"`
		PersonId    string    `json:"personId,omitempty"`
		PersonEmail string    `json:"personEmail,omitempty"`
		Created     time.Time `json:"created,omitempty"`
	} `json:"data"`
}

type MessagePostBody struct {
	RoomId   string `json:"roomId,omitempty"`
	Person   string `json:"toPersonEmail,omitempty"`
	Text     string `json:"text,omitempty"`
	Markdown string `json:"markdown,omitempty"`
}

type MessageGetResponse struct {
	Id          string    `json:"id,omitempty"`
	RoomId      string    `json:"roomId,omitempty"`
	RoomType    string    `json:"roomType,omitempty"`
	Text        string    `json:"text,omitempty"`
	Markdown    string    `json:"markdown,omitempty"`
	PersonId    string    `json:"personId,omitempty"`
	PersonEmail string    `json:"personEmail,omitempty"`
	Created     time.Time `json:"created,omitempty"`
}
