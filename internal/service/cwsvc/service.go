package cwsvc

import (
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Boards      models.BoardRepository
	Companies   models.CompanyRepository
	Contacts    models.ContactRepository
	Members     models.MemberRepository
	Tickets     models.TicketRepository
	Notes       models.TicketNoteRepository
	pool        *pgxpool.Pool
	cwClient    *psa.Client
	ticketLocks sync.Map
}

type Repos struct {
	Boards    models.BoardRepository
	Companies models.CompanyRepository
	Contacts  models.ContactRepository
	Members   models.MemberRepository
	Tickets   models.TicketRepository
	Notes     models.TicketNoteRepository
}

func New(pool *pgxpool.Pool, r models.CWRepos, cl *psa.Client) *Service {
	return &Service{
		Boards:    r.Board,
		Companies: r.Company,
		Contacts:  r.Contact,
		Members:   r.Member,
		Tickets:   r.Ticket,
		Notes:     r.Note,
		pool:      pool,
		cwClient:  cl,
	}
}

func (s *Service) withTX(tx pgx.Tx) *Service {
	return &Service{
		Boards:    s.Boards.WithTx(tx),
		Companies: s.Companies.WithTx(tx),
		Contacts:  s.Contacts.WithTx(tx),
		Members:   s.Members.WithTx(tx),
		Tickets:   s.Tickets.WithTx(tx),
		Notes:     s.Notes.WithTx(tx),
		pool:      s.pool,
		cwClient:  s.cwClient,
	}
}
