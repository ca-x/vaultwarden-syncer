package setup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/internal/service"
)

type SetupService struct {
	client      *ent.Client
	userService *service.UserService
}

type SetupData struct {
	AdminUsername string `form:"admin_username" json:"admin_username"`
	AdminPassword string `form:"admin_password" json:"admin_password"`
	AdminEmail    string `form:"admin_email" json:"admin_email,omitempty"`
}

func NewSetupService(client *ent.Client, userService *service.UserService) *SetupService {
	return &SetupService{
		client:      client,
		userService: userService,
	}
}

func (s *SetupService) IsSetupComplete(ctx context.Context) (bool, error) {
	return s.userService.HasUsers(ctx)
}

func (s *SetupService) CompleteSetup(ctx context.Context, data SetupData) error {
	hasUsers, err := s.userService.HasUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing users: %w", err)
	}

	if hasUsers {
		return fmt.Errorf("setup already completed")
	}

	if data.AdminUsername == "" {
		return fmt.Errorf("admin username is required")
	}

	if data.AdminPassword == "" {
		return fmt.Errorf("admin password is required")
	}

	if len(data.AdminPassword) < 8 {
		return fmt.Errorf("admin password must be at least 8 characters long")
	}

	_, err = s.userService.CreateUser(ctx, data.AdminUsername, data.AdminPassword, data.AdminEmail, true)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	return nil
}

func GenerateJWTSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
