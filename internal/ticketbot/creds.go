package ticketbot

import (
	"context"
	"fmt"
	"github.com/1password/onepassword-sdk-go"
	"log/slog"
)

type creds struct {
	webexSecret string
	cwPubKey    string
	cwPrivKey   string
	cwClientID  string
	cwCompanyID string
	postgresDSN string
}

const (
	webexSecretRef = "op://TicketBot/TicketBot API Credentials/webex-bot-secret"
	cwPubKeyRef    = "op://TicketBot/TicketBot API Credentials/cw-public-key"
	cwPrivKeyRef   = "op://TicketBot/TicketBot API Credentials/cw-private-key"
	cwClientIDRef  = "op://TicketBot/TicketBot API Credentials/cw-client-id"
	cwCompanyIDRef = "op://TicketBot/TicketBot API Credentials/cw-company-id"
	postgresDSNRef = "op://TicketBot/TicketBot API Credentials/postgres-dsn"
)

func new1PasswordClient(ctx context.Context, token string) (*onepassword.Client, error) {
	client, err := onepassword.NewClient(ctx, onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("TicketBot", "v1.0.0"))

	if err != nil {
		return nil, err
	}

	slog.Debug("success creating 1password client")
	return client, nil
}

func getCreds(ctx context.Context, o *onepassword.Client) (*creds, error) {
	slog.Debug("getting webex secret from 1password")
	webexSecret, err := o.Secrets().Resolve(ctx, webexSecretRef)
	if err != nil {
		return nil, fmt.Errorf("getting webex secret: %w", err)
	}

	slog.Debug("getting cw public key from 1password")
	cwPubKey, err := o.Secrets().Resolve(ctx, cwPubKeyRef)
	if err != nil {
		return nil, fmt.Errorf("getting cw public key: %w", err)
	}

	slog.Debug("getting cw private key from 1password")
	cwPrivKey, err := o.Secrets().Resolve(ctx, cwPrivKeyRef)
	if err != nil {
		return nil, fmt.Errorf("getting cw private key: %w", err)
	}

	slog.Debug("getting cw client id from 1password")
	cwClientID, err := o.Secrets().Resolve(ctx, cwClientIDRef)
	if err != nil {
		return nil, fmt.Errorf("getting cw client id: %w", err)
	}

	slog.Debug("getting cw company id from 1password")
	cwCompanyID, err := o.Secrets().Resolve(ctx, cwCompanyIDRef)
	if err != nil {
		return nil, fmt.Errorf("getting cw company id: %w", err)
	}

	slog.Debug("getting postgres dsn from 1password")
	postgresDSN, err := o.Secrets().Resolve(ctx, postgresDSNRef)
	if err != nil {
		return nil, fmt.Errorf("getting postgres dsn: %w", err)
	}

	return &creds{
		webexSecret: webexSecret,
		cwPubKey:    cwPubKey,
		cwPrivKey:   cwPrivKey,
		cwClientID:  cwClientID,
		cwCompanyID: cwCompanyID,
		postgresDSN: postgresDSN,
	}, nil
}
