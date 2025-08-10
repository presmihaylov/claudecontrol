package organizations

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type OrganizationsService struct {
	organizationsRepo *db.PostgresOrganizationsRepository
}

func NewOrganizationsService(repo *db.PostgresOrganizationsRepository) *OrganizationsService {
	return &OrganizationsService{organizationsRepo: repo}
}

func (s *OrganizationsService) CreateOrganization(ctx context.Context) (*models.Organization, error) {
	log.Printf("📋 Starting to create organization")

	organization := &models.Organization{
		ID: core.NewID("org"),
	}

	if err := s.organizationsRepo.CreateOrganization(ctx, organization); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	log.Printf("📋 Completed successfully - created organization with ID: %s", organization.ID)
	return organization, nil
}

func (s *OrganizationsService) GetOrganizationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.Organization], error) {
	log.Printf("📋 Starting to get organization by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.Organization](), fmt.Errorf("organization ID must be a valid ULID")
	}

	organization, err := s.organizationsRepo.GetOrganizationByID(ctx, id)
	if err != nil {
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by ID: %w", err)
	}

	if organization.IsPresent() {
		log.Printf("📋 Completed successfully - retrieved organization with ID: %s", id)
	} else {
		log.Printf("📋 Completed successfully - organization not found with ID: %s", id)
	}
	return organization, nil
}

func (s *OrganizationsService) GenerateCCAgentSecretKey(ctx context.Context, organizationID models.OrganizationID) (string, error) {
	log.Printf("📋 Starting to generate CCAgent secret key for organization: %s", organizationID)
	if !core.IsValidULID(string(organizationID)) {
		return "", fmt.Errorf("organization ID must be a valid ULID")
	}

	// Generate a random 32-byte secret key
	secretBytes := make([]byte, 32)
	_, err := rand.Read(secretBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random secret key: %w", err)
	}
	secretKey := base64.URLEncoding.EncodeToString(secretBytes)

	updated, err := s.organizationsRepo.GenerateCCAgentSecretKey(ctx, organizationID, secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to update organization with secret key: %w", err)
	}
	if !updated {
		return "", fmt.Errorf("organization not found")
	}

	log.Printf("📋 Completed successfully - generated secret key for organization: %s", organizationID)
	return secretKey, nil
}

func (s *OrganizationsService) GetOrganizationBySecretKey(
	ctx context.Context,
	secretKey string,
) (mo.Option[*models.Organization], error) {
	log.Printf("📋 Starting to get organization by secret key")
	if secretKey == "" {
		return mo.None[*models.Organization](), fmt.Errorf("secret key cannot be empty")
	}

	maybeOrg, err := s.organizationsRepo.GetOrganizationBySecretKey(ctx, secretKey)
	if err != nil {
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by secret key: %w", err)
	}

	if maybeOrg.IsPresent() {
		log.Printf("📋 Completed successfully - retrieved organization by secret key")
	} else {
		log.Printf("📋 Completed successfully - organization not found for secret key")
	}
	return maybeOrg, nil
}

func (s *OrganizationsService) GetAllOrganizations(ctx context.Context) ([]*models.Organization, error) {
	log.Printf("📋 Starting to get all organizations")

	organizations, err := s.organizationsRepo.GetAllOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all organizations: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d organizations", len(organizations))
	return organizations, nil
}
