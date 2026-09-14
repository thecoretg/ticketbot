package cwsvc

import (
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/repos"
)

type Service struct {
	TTL       time.Duration
	Boards    repos.BoardRepository
	Companies repos.CompanyRepository
	Contacts  repos.ContactRepository
	Members   repos.MemberRepository
	Tickets   repos.TicketRepository
	Statuses  repos.TicketStatusRepository
	Notes     repos.TicketNoteRepository
	Events    repos.TicketEventRepository
	pool      *pgxpool.Pool
	CWClient  *psa.Client

	// CWCompanyID is the ConnectWise company identifier, used to build ticket links.
	CWCompanyID string
}

func New(pool *pgxpool.Pool, r repos.CWRepos, events repos.TicketEventRepository, cl *psa.Client, companyID string, ttl int64) *Service {
	t := time.Second * time.Duration(ttl)
	return &Service{
		TTL:         t,
		Events:      events,
		CWCompanyID: companyID,
		Boards:      r.Board,
		Statuses:    r.TicketStatus,
		Companies:   r.Company,
		Contacts:    r.Contact,
		Members:     r.Member,
		Tickets:     r.Ticket,
		Notes:       r.Note,
		pool:        pool,
		CWClient:    cl,
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
		Events:    s.Events.WithTx(tx),
		pool:      s.pool,
		CWClient:  s.CWClient,

		CWCompanyID: s.CWCompanyID,
	}
}
