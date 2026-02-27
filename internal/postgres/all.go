package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/repos"
)

func AllRepos(pool *pgxpool.Pool) *repos.AllRepos {
	return &repos.AllRepos{
		APIKey:              NewAPIKeyRepo(pool),
		APIUser:             NewAPIUserRepo(pool),
		Config:              NewConfigRepo(pool),
		TicketNotifications: NewNotificationRepo(pool),
		NotifierForwards:    NewUserForwardRepo(pool),
		NotifierRules:       NewNotifierRuleRepo(pool),
		WebexRecipients:     NewWebexRecipientRepo(pool),
		CW: repos.CWRepos{
			Board:        NewBoardRepo(pool),
			TicketStatus: NewTicketStatusRepo(pool),
			Company:      NewCompanyRepo(pool),
			Contact:      NewContactRepo(pool),
			Member:       NewMemberRepo(pool),
			Note:         NewTicketNoteRepo(pool),
			Ticket:       NewTicketRepo(pool),
		},
	}
}
