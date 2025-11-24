package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/mock"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/repository/inmem"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
)

func TestNewService(t *testing.T) {
	_, err := testNewService(t, models.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
}

func TestService_ProcessNewTicket_AttemptNotifyOff(t *testing.T) {
	ctx := context.Background()
	s, err := testNewService(t, models.DefaultConfig)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	// Process as New tickets
	for _, id := range testTicketIDs(t) {
		notis, err := s.ProcessTicket(ctx, id, true)
		if err != nil {
			t.Errorf("processing new ticket %d: %v", id, err)
		}
		if len(notis) > 0 {
			t.Errorf("expected 0 notifications, got %d; ticket ID: %d", len(notis), id)
		}
	}

	// Then process again as updated tickets
	for _, id := range testTicketIDs(t) {
		notis, err := s.ProcessTicket(ctx, id, false)
		if err != nil {
			t.Errorf("processing updated ticket %d: %v", id, err)
		}
		if len(notis) > 0 {
			t.Errorf("expected 0 notifications, got %d; ticket ID: %d", len(notis), id)
		}
	}
}

func TestService_ProcessNewTicket_AttemptNotifyOn(t *testing.T) {
	ctx := context.Background()
	cfg := &models.DefaultConfig
	cfg.AttemptNotify = true

	s, err := testNewService(t, *cfg)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	if err := testSeedNotifiers(t, ctx, s); err != nil {
		t.Fatalf("adding test notifiers: %v", err)
	}

	// Process as New tickets
	for _, id := range testTicketIDs(t) {
		_, err := s.ProcessTicket(ctx, id, true)
		if err != nil {
			t.Errorf("processing new ticket %d: %v", id, err)
		}
	}

	existing, err := s.Notifier.Notifications.ListAll(ctx)
	if err != nil {
		t.Fatalf("listing existing notifications: %v", err)
	}

	for _, e := range existing {
		t.Logf("existing notification found with note %d: %v", e.TicketNoteID, e)
	}

	// Then process again as updated tickets
	for _, id := range testTicketIDs(t) {
		_, err := s.ProcessTicket(ctx, id, false)
		if err != nil {
			t.Errorf("processing updated ticket %d: %v", id, err)
		}
	}
}

func testNewService(t *testing.T, cfg models.Config) (*Service, error) {
	t.Helper()
	slog.SetLogLoggerLevel(slog.LevelDebug)
	if err := godotenv.Load("testing.env"); err != nil {
		return nil, fmt.Errorf("loading .env")
	}

	cwCreds := testGetCwCreds(t)
	webexSecret := os.Getenv("WEBEX_SECRET")

	cwRepos := models.CWRepos{
		Board:   inmem.NewBoardRepo(nil),
		Company: inmem.NewCompanyRepo(nil),
		Contact: inmem.NewContactRepo(nil),
		Member:  inmem.NewMemberRepo(nil),
		Note:    inmem.NewTicketNoteRepo(nil),
		Ticket:  inmem.NewTicketRepo(nil),
	}

	notiRepos := notifier.Repos{
		Rooms:         inmem.NewWebexRoomRepo(nil),
		Notifiers:     inmem.NewNotifierRepo(nil),
		Notifications: inmem.NewNotificationRepo(nil),
		Forwards:      inmem.NewUserForwardRepo(nil),
	}

	cs := cwsvc.New(nil, cwRepos, psa.NewClient(cwCreds))
	ns := notifier.New(cfg, notiRepos, mock.NewWebexClient(webexSecret), cwCreds.CompanyId)

	return New(models.DefaultConfig, cs, ns), nil
}

func testGetCwCreds(t *testing.T) *psa.Creds {
	t.Helper()
	return &psa.Creds{
		PublicKey:  os.Getenv("CW_PUB_KEY"),
		PrivateKey: os.Getenv("CW_PRIV_KEY"),
		ClientId:   os.Getenv("CW_CLIENT_ID"),
		CompanyId:  os.Getenv("CW_COMPANY_ID"),
	}
}

func testTicketIDs(t *testing.T) []int {
	t.Helper()
	raw := os.Getenv("TEST_TICKET_IDS")
	split := strings.Split(raw, ",")

	var ids []int
	for _, s := range split {
		i, err := strconv.Atoi(s)
		if err != nil {
			t.Logf("couldn't convert ticket id '%s' to integer", s)
			continue
		}
		ids = append(ids, i)
	}

	return ids
}

func testSeedNotifiers(t *testing.T, ctx context.Context, s *Service) error {
	r := models.WebexRoom{
		WebexID: "something",
		Name:    "something",
		Type:    "group",
	}

	room, err := s.Notifier.Rooms.Upsert(ctx, r)
	if err != nil {
		return fmt.Errorf("inserting mock room: %w", err)
	}

	for _, b := range testBoardIDs(t) {

		n := &models.Notifier{
			CwBoardID:     b,
			WebexRoomID:   room.ID,
			NotifyEnabled: true,
		}

		if _, err := s.Notifier.Notifiers.Insert(ctx, n); err != nil {
			return err
		}
	}

	return nil
}

func testBoardIDs(t *testing.T) []int {
	raw := os.Getenv("TEST_ENABLED_BOARD_IDS")
	split := strings.Split(raw, ",")

	var ids []int
	for _, s := range split {
		i, err := strconv.Atoi(s)
		if err != nil {
			t.Logf("couldn't convert board id '%s' to integer", s)
			continue
		}
		ids = append(ids, i)
	}

	return ids
}
