package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"tctg-automation/pkg/connectwise"
)

func (s *server) getAndStoreResourceEmails(ctx context.Context, resourceString string, mutedUsers []string) ([]string, error) {
	ids := splitTicketResources(resourceString)
	if ids == nil {
		return nil, nil // No resources to process
	}

	var emails []string
	for _, id := range ids {
		if isMuted(id, mutedUsers) {
			slog.Debug("user muted per calling function", "id", id, "mutedUsers", mutedUsers)
			continue
		}

		id = strings.TrimSpace(id)
		if id == "" {
			slog.Debug("skipping empty resource ID", "id", id)
			continue // Skip empty IDs
		}

		u, err := getUserByCwID(s.db, id)
		if err != nil {
			slog.Error("failed to get user by ConnectWise ID", "id", id, "error", err)
			continue
		}

		if u != nil {
			if u.Email == "" {
				slog.Debug("user found in database but has no email", "id", id)
				continue // Skip users without an email, no need for mute check
			}

			slog.Debug("found user in database, using cached email", "id", id, "email", u.Email)
			if u.Mute {
				slog.Debug("user is marked as muted, skipping", "id", id, "email", u.Email)
				continue // Skip excluded users
			}

			emails = append(emails, u.Email)
			continue // Use cached email if available
		}

		slog.Debug("user not found in database, fetching email from ConnectWise", "id", id)
		email, err := s.getMemberEmail(ctx, id)
		if err != nil {
			slog.Error("failed to get member email", "id", id, "error", err)
			continue
		}

		if email == "" {
			slog.Debug("no email found for member", "id", id)
			continue // Skip if no email is found
		}

		newUser := &user{
			CWId:  id,
			Email: email,
			Mute:  false, // Default to not excluded
		}

		if _, err := addOrUpdateUser(s.db, newUser); err != nil {
			slog.Error("failed to add or update user in database", "id", id, "email", email, "error", err)
			continue
		}

		slog.Info("added or updated user in database", "id", id, "email", email)

		emails = append(emails, email)
	}

	return emails, nil
}

func (s *server) getMemberEmail(ctx context.Context, id string) (string, error) {
	q := &connectwise.QueryParams{
		Conditions: fmt.Sprintf("Identifier='%s'", id),
	}

	m, err := s.cwClient.ListMembers(ctx, q)
	if err != nil {
		return "", fmt.Errorf("getting member with query: %w", err)
	}

	if len(m) == 0 {
		return "", nil
	}

	if len(m) > 1 {
		return "", fmt.Errorf("too many members (%d) returned for id %s", len(m), id)
	}

	if m[0].PrimaryEmail == "" {
		return "", fmt.Errorf("empty email found for member id %s", id)
	}

	return m[0].PrimaryEmail, nil
}

func splitTicketResources(resourceString string) []string {
	parts := strings.Split(resourceString, ",")
	var resources []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		resources = append(resources, part)
	}

	if len(resources) == 0 {
		return nil
	} else {
		return resources
	}
}
