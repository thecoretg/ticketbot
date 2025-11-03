package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (cl *Client) ensureMemberByIdentifier(ctx context.Context, identifier string) (db.CwMember, error) {
	member, err := cl.Queries.GetMemberByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("member not in store, attempting insert", "member_identifier", identifier)
			cwMember, err := cl.CWClient.GetMemberByIdentifier(identifier)
			if err != nil {
				return db.CwMember{}, fmt.Errorf("getting member from cw by identifier: %w", err)
			}

			if cwMember == nil {
				return db.CwMember{}, fmt.Errorf("member %s not found", identifier)
			}

			return cl.ensureMemberInStore(ctx, cwMember.ID)
		}
		return db.CwMember{}, fmt.Errorf("querying db for member: %w", err)
	}

	return member, nil
}

func (cl *Client) ensureMemberInStore(ctx context.Context, memberID int) (db.CwMember, error) {
	member, err := cl.Queries.GetMember(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			cwMember, err := cl.CWClient.GetMember(memberID, nil)
			if err != nil {
				return db.CwMember{}, fmt.Errorf("getting member from cw: %w", err)
			}
			p := db.UpsertMemberParams{
				ID:           memberID,
				Identifier:   cwMember.Identifier,
				FirstName:    cwMember.FirstName,
				LastName:     cwMember.LastName,
				PrimaryEmail: cwMember.PrimaryEmail,
			}

			member, err = cl.Queries.UpsertMember(ctx, p)
			if err != nil {
				return db.CwMember{}, fmt.Errorf("inserting member into db: %w", err)
			}
			return member, nil
		} else {
			return db.CwMember{}, fmt.Errorf("getting member from db: %w", err)
		}
	}

	return member, nil
}
