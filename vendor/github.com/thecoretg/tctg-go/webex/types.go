package webex

import (
	"encoding/json"
	"time"
)

type ListWebhooksResp struct {
	Items []Webhook `json:"items"`
}

type Webhook struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	Resource  string `json:"resource"`
	Event     string `json:"event"`
	Filter    string `json:"filter,omitempty"`
	Secret    string `json:"secret,omitempty"`
}

type MessageHookPayload struct {
	Data Message `json:"data"`
}

type Message struct {
	ID          string            `json:"id,omitempty"`
	RoomID      string            `json:"roomId,omitempty"`
	RoomType    string            `json:"roomType,omitempty"`
	Text        string            `json:"text,omitempty"`
	Markdown    string            `json:"markdown,omitempty"`
	Attachments []json.RawMessage `json:"attachments,omitempty"`

	// Use ToPersonEmail for posts. PersonEmail (no to) is returned in gets.
	// Same with ToPersonId and PersonId.
	// I don't make the rules. Thanks Webex <3
	ToPersonEmail string `json:"toPersonEmail,omitempty"`
	ToPersonID    string `json:"toPersonId,omitempty"`
	PersonEmail   string `json:"personEmail,omitempty"`
	PersonID      string `json:"personId,omitempty"`

	// This does not go to the request; it is used for logging in the server message requests,
	// specifically for rooms so the name can be identified
	// TODO: get rid of these
	RecipientType string
	RecipientName string
}

type AttachmentAction struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	MessageID string            `json:"messageId"`
	Inputs    map[string]string `json:"inputs"`
	PersonID  string            `json:"personId"`
	RoomID    string            `json:"roomId"`
	Created   time.Time         `json:"created"`
}

type ListRoomsResp struct {
	Items []Room `json:"items"`
}

type Room struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Type         string    `json:"type"`
	IsLocked     bool      `json:"isLocked"`
	LastActivity time.Time `json:"lastActivity"`
	CreatorID    string    `json:"creatorId"`
	Created      time.Time `json:"created"`
	OwnerID      string    `json:"ownerId"`
	IsPublic     bool      `json:"isPublic"`
	IsReadOnly   bool      `json:"isReadOnly"`
}

type ListPeopleResp struct {
	NotFoundIds any      `json:"notFoundIds"`
	Items       []Person `json:"items"`
}

type Person struct {
	ID           string   `json:"id"`
	Emails       []string `json:"emails"`
	PhoneNumbers []struct {
		Type    string `json:"type"`
		Value   string `json:"value"`
		Primary bool   `json:"primary"`
	} `json:"phoneNumbers"`
	DisplayName  string    `json:"displayName"`
	NickName     string    `json:"nickName"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	Avatar       string    `json:"avatar"`
	OrgID        string    `json:"orgId"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	LastActivity time.Time `json:"lastActivity"`
	Status       string    `json:"status"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
}
