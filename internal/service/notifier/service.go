package notifier

import (
	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Rooms     models.WebexRoomRepository
	Notifiers models.NotifierRepository
	Forwards  models.UserForwardRepository
}
