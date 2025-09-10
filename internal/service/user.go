package service

import (
	"context"
	"fmt"
	"vaultwarden-syncer/ent"
	"vaultwarden-syncer/ent/user"
	"vaultwarden-syncer/internal/auth"
)

type UserService struct {
	client *ent.Client
	auth   *auth.Service
}

func NewUserService(client *ent.Client, auth *auth.Service) *UserService {
	return &UserService{
		client: client,
		auth:   auth,
	}
}

func (s *UserService) CreateUser(ctx context.Context, username, password, email string, isAdmin bool) (*ent.User, error) {
	hashedPassword, err := s.auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	u, err := s.client.User.
		Create().
		SetUsername(username).
		SetPassword(hashedPassword).
		SetEmail(email).
		SetIsAdmin(isAdmin).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return u, nil
}

func (s *UserService) Authenticate(ctx context.Context, username, password string) (string, *ent.User, error) {
	u, err := s.client.User.
		Query().
		Where(user.Username(username)).
		Only(ctx)

	if err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	if !s.auth.VerifyPassword(password, u.Password) {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	token, err := s.auth.GenerateToken(u.ID, u.Username, u.IsAdmin)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, u, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id int) (*ent.User, error) {
	return s.client.User.Get(ctx, id)
}

func (s *UserService) HasUsers(ctx context.Context) (bool, error) {
	count, err := s.client.User.Query().Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}