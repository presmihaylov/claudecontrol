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
	organizationsService services.OrganizationsService
	txManager            services.TransactionManager
}

func NewUsersService(
	repo *db.PostgresUsersRepository,
	organizationsService services.OrganizationsService,
	txManager services.TransactionManager,
) *UsersService {
	return &UsersService{
		usersRepo:            repo,
		organizationsService: organizationsService,
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
		// Get or create the user
		user, err := s.usersRepo.GetOrCreateUser(txCtx, authProvider, authProviderID)
		if err != nil {
			return fmt.Errorf("failed to get or create user: %w", err)
		}

		// Check if user needs an organization
		if user.OrganizationID == "" {
			log.Printf("📋 User %s needs an organization, creating one", user.ID)

			// Create organization
			organization, err := s.organizationsService.CreateOrganization(txCtx)
			if err != nil {
				return fmt.Errorf("failed to create organization: %w", err)
			}

			// Update user's organization_id
			err = s.usersRepo.UpdateUserOrganization(txCtx, user.ID, organization.ID)
			if err != nil {
				return fmt.Errorf("failed to update user organization: %w", err)
			}

			// Update user struct with organization_id
			user.OrganizationID = organization.ID
		}

		finalUser = user
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get or create user with organization: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved/created user with ID: %s", finalUser.ID)
	return finalUser, nil
}
