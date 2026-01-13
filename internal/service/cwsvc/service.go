package cwsvc

import (
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

type Service struct {
	TTL       time.Duration
	Boards    models.BoardRepository
	Companies models.CompanyRepository
	Contacts  models.ContactRepository
	Members   models.MemberRepository
	Tickets   models.TicketRepository
	Statuses  models.TicketStatusRepository
	Notes     models.TicketNoteRepository
	pool      *pgxpool.Pool
	CWClient  *psa.Client
}

func New(pool *pgxpool.Pool, r models.CWRepos, cl *psa.Client, ttl int64) *Service {
	t := time.Second * time.Duration(ttl)
	return &Service{
		TTL:       t,
		Boards:    r.Board,
		Statuses:  r.TicketStatus,
		Companies: r.Company,
		Contacts:  r.Contact,
		Members:   r.Member,
		Tickets:   r.Ticket,
		Notes:     r.Note,
		pool:      pool,
		CWClient:  cl,
	}
}

func (s *Service) WithTX(tx pgx.Tx) *Service {
	return &Service{
		TTL:       s.TTL,
		Boards:    s.Boards.WithTx(tx),
		Statuses:  s.Statuses.WithTx(tx),
		Companies: s.Companies.WithTx(tx),
		Contacts:  s.Contacts.WithTx(tx),
		Members:   s.Members.WithTx(tx),
		Tickets:   s.Tickets.WithTx(tx),
		Notes:     s.Notes.WithTx(tx),
		pool:      s.pool,
		CWClient:  s.CWClient,
	}
}
