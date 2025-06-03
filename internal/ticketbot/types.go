package ticketbot

type boardSetting struct {
	BoardID     int    `json:"board_id"`
	BoardName   string `json:"board_name"`
	WebexRoomID string `json:"webex_room_id"`
}
