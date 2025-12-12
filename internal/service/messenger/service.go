package messenger

import (
	"context"
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/user"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

var ErrNotAnAdmin = errors.New("no admin user found")

type Service struct {
	UserSvc  *user.Service
	WebexSvc *webexsvc.Service
}

func New(wx *webexsvc.Service) *Service {
	return &Service{WebexSvc: wx}
}

func (s *Service) ParseAndRespond(ctx context.Context, msg *webex.Message) error {
	if msg == nil {
		return errors.New("got nil message")
	}

	m, err := s.parseIncoming(ctx, msg)
	if err != nil {
		return fmt.Errorf("parsing incoming message: %w", err)
	}

	if m == nil {
		return errors.New("parse returned nil message")
	}

	if _, err := s.WebexSvc.PostMessage(m); err != nil {
		return fmt.Errorf("posting webex message: %w", err)
	}
	return nil
}

func (s *Service) parseIncoming(ctx context.Context, msg *webex.Message) (*webex.Message, error) {
	email := msg.PersonEmail
	if email == "" {
		return nil, errors.New("got empty email field")
	}

	var m *webex.Message
	if _, err := s.getValidUser(ctx, email); err != nil {
		if errors.Is(err, ErrNotAnAdmin) {
			return notAnAdminMessage(email), nil
		}
	}

	return m, nil
}

func (s *Service) getValidUser(ctx context.Context, email string) (*models.APIUser, error) {
	u, err := s.UserSvc.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			return nil, ErrNotAnAdmin
		}
		return nil, fmt.Errorf("getting user by email: %w", err)
	}

	return u, nil
}

func notAnAdminMessage(email string) *webex.Message {
	txt := "Sorry, this command requires admin permissions!"
	m := webex.NewMessageToPerson(email, txt)
	return &m
}
