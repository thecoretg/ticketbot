package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/ticketbot/internal/repos"
)

type APIUserRepo struct {
	queries *db.Queries
}

func NewAPIUserRepo(pool *pgxpool.Pool) *APIUserRepo {
	return &APIUserRepo{queries: db.New(pool)}
}

func (p *APIUserRepo) WithTx(tx pgx.Tx) repos.APIUserRepository {
	return &APIUserRepo{queries: db.New(tx)}
}

func (p *APIUserRepo) List(ctx context.Context) ([]*models.APIUser, error) {
	dk, err := p.queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	var k []*models.APIUser
	for _, d := range dk {
		u := userFromPG(d)
		k = append(k, u)
	}

	return k, nil
}

func (p *APIUserRepo) Get(ctx context.Context, id int) (*models.APIUser, error) {
	d, err := p.queries.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrAPIUserNotFound
		}
		return nil, models.ErrAPIUserNotFound
	}

	return userFromPG(d), nil
}

func (p *APIUserRepo) GetByEmail(ctx context.Context, email string) (*models.APIUser, error) {
	d, err := p.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrAPIUserNotFound
		}
		return nil, err
	}

	return userFromPG(d), nil
}

func (p *APIUserRepo) Exists(ctx context.Context, email string) (bool, error) {
	return p.queries.CheckUserExists(ctx, email)
}

func (p *APIUserRepo) Insert(ctx context.Context, email string) (*models.APIUser, error) {
	d, err := p.queries.InsertUser(ctx, email)
	if err != nil {
		return nil, err
	}

	return userFromPG(d), nil
}

func (p *APIUserRepo) Update(ctx context.Context, u *models.APIUser) (*models.APIUser, error) {
	d, err := p.queries.UpdateUser(ctx, apiUserToUpdateParams(u))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrAPIUserNotFound
		}
		return nil, err
	}

	return userFromPG(d), nil
}

func (p *APIUserRepo) GetForAuth(ctx context.Context, email string) (*models.UserAuth, error) {
	d, err := p.queries.GetUserForAuth(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrAPIUserNotFound
		}
		return nil, err
	}

	return userAuthFromPG(d.ID, d.EmailAddress, d.PasswordHash, d.PasswordResetRequired, d.TotpSecret, d.TotpEnabled), nil
}

func (p *APIUserRepo) GetForAuthByID(ctx context.Context, id int) (*models.UserAuth, error) {
	d, err := p.queries.GetUserForAuthByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrAPIUserNotFound
		}
		return nil, err
	}

	return userAuthFromPG(d.ID, d.EmailAddress, d.PasswordHash, d.PasswordResetRequired, d.TotpSecret, d.TotpEnabled), nil
}

func (p *APIUserRepo) SetPassword(ctx context.Context, id int, hash []byte) error {
	return p.queries.SetUserPassword(ctx, db.SetUserPasswordParams{
		ID:           id,
		PasswordHash: hash,
	})
}

func (p *APIUserRepo) SetTOTPSecret(ctx context.Context, id int, secret *string) error {
	return p.queries.SetTOTPSecret(ctx, db.SetTOTPSecretParams{ID: id, TotpSecret: secret})
}

func (p *APIUserRepo) SetTOTPEnabled(ctx context.Context, id int, enabled bool) error {
	return p.queries.SetTOTPEnabled(ctx, db.SetTOTPEnabledParams{ID: id, TotpEnabled: enabled})
}

func (p *APIUserRepo) SetPasswordResetRequired(ctx context.Context, id int, required bool) error {
	return p.queries.SetPasswordResetRequired(ctx, db.SetPasswordResetRequiredParams{
		ID:                    id,
		PasswordResetRequired: required,
	})
}

func (p *APIUserRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteUser(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAPIUserNotFound
		}
		return err
	}

	return nil
}

func apiUserToUpdateParams(u *models.APIUser) db.UpdateUserParams {
	return db.UpdateUserParams{
		ID:           u.ID,
		EmailAddress: u.EmailAddress,
	}
}

func userFromPG(pg *db.ApiUser) *models.APIUser {
	return &models.APIUser{
		ID:           pg.ID,
		EmailAddress: pg.EmailAddress,
		CreatedOn:    pg.CreatedOn,
		UpdatedOn:    pg.UpdatedOn,
	}
}

func userAuthFromPG(id int, email string, passwordHash []byte, resetRequired bool, totpSecret *string, totpEnabled bool) *models.UserAuth {
	return &models.UserAuth{
		ID:            id,
		EmailAddress:  email,
		PasswordHash:  passwordHash,
		ResetRequired: resetRequired,
		TOTPSecret:    totpSecret,
		TOTPEnabled:   totpEnabled,
	}
}
