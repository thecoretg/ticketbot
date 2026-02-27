package webexsvc

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/repos"
)

type Service struct {
	Recipients  repos.WebexRecipientRepository
	pool        *pgxpool.Pool
	WebexClient repos.MessageSender
	BotEmail    string
}

func New(pool *pgxpool.Pool, r repos.WebexRecipientRepository, cl repos.MessageSender, botEmail string) *Service {
	return &Service{
		Recipients:  r,
		WebexClient: cl,
		BotEmail:    botEmail,
		pool:        pool,
	}
}

func (s *Service) WithTx(tx pgx.Tx) *Service {
	return &Service{
		Recipients:  s.Recipients.WithTx(tx),
		WebexClient: s.WebexClient,
		BotEmail:    s.BotEmail,
		pool:        s.pool,
	}
}
