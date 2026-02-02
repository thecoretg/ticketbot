package syncsvc

import (
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
)

type Service struct {
	CW       *cwsvc.Service
	Webex    *webexsvc.Service
	Notifier *notifier.Service
	pool     *pgxpool.Pool
	syncing  atomic.Bool
}

func New(pool *pgxpool.Pool, cw *cwsvc.Service, wx *webexsvc.Service, ns *notifier.Service) *Service {
	return &Service{
		CW:       cw,
		Webex:    wx,
		Notifier: ns,
		pool:     pool,
	}
}

func (s *Service) withTx(tx pgx.Tx) *Service {
	return &Service{
		CW:    s.CW.WithTX(tx),
		Webex: s.Webex.WithTx(tx),
		pool:  s.pool,
	}
}
