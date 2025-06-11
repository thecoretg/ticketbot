package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"tctg-automation/pkg/connectwise"
)

func (s *Server) getAndCacheResourceEmails(ctx context.Context, resourceString string, exclusions []string) ([]string, error) {
	ids := splitTicketResources(resourceString)
	if ids == nil {
		return nil, nil // No resources to process
	}

	var emails []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			slog.Debug("skipping empty resource ID", "id", id)
			continue // Skip empty IDs
		}

		if isExcluded(id, exclusions) {
			slog.Debug("skipping excluded resource", "id", id)
			continue // Skip excluded IDs
		}

		if e, exists := s.users[id]; exists {
			// If the email is already cached, use it
			slog.Debug("using cached email for resource", "id", id, "email", e)
			emails = append(emails, e)
			continue
		}

		// Fetch the member's email and cache it
		email, err := s.getMemberEmail(ctx, id)
		if err != nil {
			slog.Error("failed to get member email", "id", id, "error", err)
			return nil, fmt.Errorf("getting member email for id %s: %w", id, err)
		}

		if email != "" {
			slog.Info("caching member email", "id", id, "email", email)
			s.users[id] = email
		}

		emails = append(emails, email)
	}

	return emails, nil
}

func (s *Server) getMemberEmail(ctx context.Context, id string) (string, error) {
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

func isExcluded(id string, exclusions []string) bool {
	for _, e := range exclusions {
		if e == id {
			return true
		}
	}
	return false
}
