package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

func AllRepos(pool *pgxpool.Pool) *models.AllRepos {
	return &models.AllRepos{
		APIKey:        NewAPIKeyRepo(pool),
		APIUser:       NewAPIUserRepo(pool),
		Config:        NewConfigRepo(pool),
		Notifications: NewNotificationRepo(pool),
		Forwards:      NewUserForwardRepo(pool),
		NotifierRules: NewNotifierRuleRepo(pool),
		WebexRoom:     NewWebexRoomRepo(pool),
		CW: models.CWRepos{
			Board:   NewBoardRepo(pool),
			Company: NewCompanyRepo(pool),
			Contact: NewContactRepo(pool),
			Member:  NewMemberRepo(pool),
			Note:    NewTicketNoteRepo(pool),
			Ticket:  NewTicketRepo(pool),
		},
	}
}
