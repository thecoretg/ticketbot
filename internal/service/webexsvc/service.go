package webexsvc

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/external/webex"
	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Rooms       models.WebexRoomRepository
	pool        *pgxpool.Pool
	webexClient *webex.Client
}

func New(pool *pgxpool.Pool, r models.WebexRoomRepository, cl *webex.Client) *Service {
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
