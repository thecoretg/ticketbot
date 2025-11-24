package webexsvc

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Rooms       models.WebexRoomRepository
	pool        *pgxpool.Pool
	webexClient models.MessageSender
}

func New(pool *pgxpool.Pool, r models.WebexRoomRepository, cl models.MessageSender) *Service {
	return &Service{
		Rooms:       r,
		pool:        pool,
		webexClient: cl,
	}
}

func (s *Service) withTx(tx pgx.Tx) *Service {
	return &Service{
		Rooms:       s.Rooms.WithTx(tx),
		pool:        s.pool,
		webexClient: s.webexClient,
	}
}
