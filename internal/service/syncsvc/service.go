package syncsvc

import (
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
)

type Service struct {
	CW        *cwsvc.Service
	Webex     *webexsvc.Service
	Ticketbot *ticketbot.Service
	pool      *pgxpool.Pool
	syncing   atomic.Bool
}

func New(pool *pgxpool.Pool, cw *cwsvc.Service, wx *webexsvc.Service, tb *ticketbot.Service) *Service {
	return &Service{
		CW:        cw,
		Webex:     wx,
		Ticketbot: tb,
		pool:      pool,
	}
}

func (s *Service) withTx(tx pgx.Tx) *Service {
	return &Service{
		CW:    s.CW.WithTX(tx),
		Webex: s.Webex.WithTx(tx),
		pool:  s.pool,
	}
}
