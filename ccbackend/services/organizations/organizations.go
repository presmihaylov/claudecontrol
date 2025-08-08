package organizations

import (
	"context"
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

	maybeOrg, err := s.organizationsRepo.GetOrganizationByID(ctx, id)
	if err != nil {
		return mo.None[*models.Organization](), fmt.Errorf("failed to get organization by ID: %w", err)
	}

	if !maybeOrg.IsPresent() {
		log.Printf("📋 Completed successfully - organization not found")
		return mo.None[*models.Organization](), nil
	}

	organization := maybeOrg.MustGet()
	log.Printf("📋 Completed successfully - found organization: %s", organization.ID)
	return mo.Some(organization), nil
}
