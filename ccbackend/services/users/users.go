package users

import (
	"context"
	"fmt"
	"log"

	"ccbackend/db"
	"ccbackend/models"
)

type UsersService struct {
	usersRepo *db.PostgresUsersRepository
}

func NewUsersService(repo *db.PostgresUsersRepository) *UsersService {
	return &UsersService{usersRepo: repo}
}

func (s *UsersService) GetOrCreateUser(ctx context.Context, authProvider, authProviderID string) (*models.User, error) {
	log.Printf("📋 Starting to get or create user for authProvider: %s, authProviderID: %s", authProvider, authProviderID)

	if authProvider == "" {
		return nil, fmt.Errorf("auth_provider cannot be empty")
	}

	if authProviderID == "" {
		return nil, fmt.Errorf("auth_provider_id cannot be empty")
	}

	user, err := s.usersRepo.GetOrCreateUser(ctx, authProvider, authProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved/created user with ID: %s", user.ID)
	return user, nil
}
