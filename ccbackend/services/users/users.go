package users

import (
	"context"
	"fmt"
	"log"

	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services"
)

type UsersService struct {
	usersRepo            *db.PostgresUsersRepository
	organisationsService services.OrganisationsService
	txManager            services.TransactionManager
}

func NewUsersService(
	repo *db.PostgresUsersRepository,
	organisationsService services.OrganisationsService,
	txManager services.TransactionManager,
) *UsersService {
	return &UsersService{
		usersRepo:            repo,
		organisationsService: organisationsService,
		txManager:            txManager,
	}
}

func (s *UsersService) GetOrCreateUser(ctx context.Context, authProvider, authProviderID string) (*models.User, error) {
	log.Printf(
		"📋 Starting to get or create user for authProvider: %s, authProviderID: %s",
		authProvider,
		authProviderID,
	)
	if authProvider == "" {
		return nil, fmt.Errorf("auth_provider cannot be empty")
	}
	if authProviderID == "" {
		return nil, fmt.Errorf("auth_provider_id cannot be empty")
	}

	var finalUser *models.User
	err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
		// First, try to get existing user with SELECT FOR UPDATE
		existingUser, err := s.usersRepo.GetUserByAuthProvider(txCtx, authProvider, authProviderID, true)
		if err != nil {
			return fmt.Errorf("failed to get existing user: %w", err)
		}

		// If user exists, return them
		if existingUser != nil {
			finalUser = existingUser
			return nil
		}

		// User doesn't exist, so create organization first
		organization, err := s.organisationsService.CreateOrganisation(txCtx)
		if err != nil {
			return fmt.Errorf("failed to create organization: %w", err)
		}

		// Create new user with organization_id
		newUser, err := s.usersRepo.CreateUser(txCtx, authProvider, authProviderID, organization.ID)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		finalUser = newUser
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get or create user with organization: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved/created user with ID: %s", finalUser.ID)
	return finalUser, nil
}
