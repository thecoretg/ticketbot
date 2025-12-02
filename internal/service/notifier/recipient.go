package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
)

var ErrNoRoomsForEmail = errors.New("no rooms found for this email")

func (s *Service) getRecipientRoomIDs(ctx context.Context, ticket *models.FullTicket) []int {
	var excluded []models.Member

	// if the sender of the note is a member, exclude them from messages;
	// they don't need a notification for their own note
	if ticket.LatestNote != nil && ticket.LatestNote.Member != nil {
		excluded = append(excluded, *ticket.LatestNote.Member)
	}

	var roomIDs []int
	for _, r := range ticket.Resources {
		if memberSliceContains(excluded, r) {
			continue
		}

		e, err := s.processFwds(ctx, r.PrimaryEmail)
		if err != nil {
			slog.Error("notifier: error checking forwards for email", "ticket_id", ticket.Ticket.ID, "email", r.PrimaryEmail, "error", err)
		}

		roomIDs = append(roomIDs, e...)
	}

	return filterDuplicateEmails(roomIDs)
}

func (s *Service) processFwds(ctx context.Context, email string) ([]int, error) {
	rooms, err := s.Rooms.ListByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("listing webex rooms by email %s: %w", email, err)
	}

	if len(rooms) == 0 {
		return nil, ErrNoRoomsForEmail
	}

	var src models.WebexRecipient
	if len(rooms) > 1 {
		sort.Slice(rooms, func(i, j int) bool {
			return rooms[i].LastActivity.After(rooms[j].LastActivity)
		})
	}
	src = rooms[0]

	noFwds := []int{src.ID}
	fwds, err := s.Forwards.ListBySourceRoomID(ctx, src.ID)
	if err != nil {
		return noFwds, fmt.Errorf("checking forwards: %w", err)
	}

	if len(fwds) == 0 {
		return noFwds, nil
	}

	activeFwds := filterActiveFwds(fwds)
	if len(activeFwds) == 0 {
		return noFwds, nil
	}

	toNotify := make(map[int]struct{})
	keep := false
	for _, f := range activeFwds {
		if f.UserKeepsCopy {
			keep = true
		}

		toNotify[f.DestID] = struct{}{}
	}

	if keep {
		toNotify[src.ID] = struct{}{}
	}

	var tn []int
	for k := range toNotify {
		tn = append(tn, k)
	}

	return tn, nil
}

func memberSliceContains(members []models.Member, check models.Member) bool {
	for _, x := range members {
		if x.ID == check.ID {
			return true
		}
	}

	return false
}

func filterActiveFwds(fwds []models.NotifierForward) []models.NotifierForward {
	var activeFwds []models.NotifierForward
	for _, f := range fwds {
		if f.Enabled && dateRangeActive(f.StartDate, f.EndDate) {
			activeFwds = append(activeFwds, f)
		}
	}

	return activeFwds
}

func dateRangeActive(start, end *time.Time) bool {
	now := time.Now()
	if start == nil {
		return false
	}

	if end == nil {
		return now.After(*start)
	}

	return now.After(*start) && now.Before(*end)
}

func filterDuplicateEmails(ids []int) []int {
	seenIDs := make(map[int]struct{})
	for _, i := range ids {
		if _, ok := seenIDs[i]; !ok {
			seenIDs[i] = struct{}{}
		}
	}

	var uniqueIDs []int
	for e := range seenIDs {
		uniqueIDs = append(uniqueIDs, e)
	}

	return uniqueIDs
}
