package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type NotificationRepo struct {
	queries *db.Queries
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{queries: db.New(pool)}
}

func (p NotificationRepo) WithTx(tx pgx.Tx) models.TicketNotificationRepository {
	return &NotificationRepo{queries: db.New(tx)}
}

func (p NotificationRepo) ListAll(ctx context.Context) ([]*models.TicketNotification, error) {
	dn, err := p.queries.ListTicketNotifications(ctx)
	if err != nil {
		return nil, err
	}

	var n []*models.TicketNotification
	for _, d := range dn {
		n = append(n, notificationFromPG(d))
	}

	return n, nil
}

func (p NotificationRepo) ListByNoteID(ctx context.Context, noteID int) ([]*models.TicketNotification, error) {
	dn, err := p.queries.ListTicketNotificationsByNoteID(ctx, &noteID)
	if err != nil {
		return nil, err
	}

	var n []*models.TicketNotification
	for _, d := range dn {
		n = append(n, notificationFromPG(d))
	}

	return n, nil
}

func (p NotificationRepo) ExistsForTicket(ctx context.Context, ticketID int) (bool, error) {
	exists, err := p.queries.CheckNotificationsExistByTicketID(ctx, ticketID)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (p NotificationRepo) ExistsForNote(ctx context.Context, noteID int) (bool, error) {
	exists, err := p.queries.CheckNotificationsExistByNote(ctx, &noteID)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (p NotificationRepo) Get(ctx context.Context, id int) (*models.TicketNotification, error) {
	d, err := p.queries.GetTicketNotification(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotificationNotFound
		}
		return nil, nil
	}

	return notificationFromPG(d), nil
}

func (p NotificationRepo) Insert(ctx context.Context, n *models.TicketNotification) (*models.TicketNotification, error) {
	d, err := p.queries.InsertTicketNotification(ctx, notificationToInsertParams(n))
	if err != nil {
		return nil, err
	}

	return notificationFromPG(d), nil
}

func (p NotificationRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicketNotification(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrNotificationNotFound
		}
		return err
	}

	return nil
}

func notificationToInsertParams(n *models.TicketNotification) db.InsertTicketNotificationParams {
	return db.InsertTicketNotificationParams{
		TicketID:        n.TicketID,
		TicketNoteID:    n.TicketNoteID,
		RecipientID:     n.RecipientID,
		ForwardedFromID: n.ForwardedFromID,
		Sent:            n.Sent,
		Skipped:         n.Skipped,
	}
}

func notificationFromPG(pg *db.TicketNotification) *models.TicketNotification {
	return &models.TicketNotification{
		ID:              pg.ID,
		TicketID:        pg.TicketID,
		TicketNoteID:    pg.TicketNoteID,
		RecipientID:     pg.RecipientID,
		ForwardedFromID: pg.ForwardedFromID,
		Sent:            pg.Sent,
		Skipped:         pg.Skipped,
		CreatedOn:       pg.CreatedOn,
		UpdatedOn:       pg.UpdatedOn,
	}
}
