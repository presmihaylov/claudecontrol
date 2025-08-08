package organisations

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type OrganisationsService struct {
	organisationsRepo *db.PostgresOrganisationsRepository
}

func NewOrganisationsService(repo *db.PostgresOrganisationsRepository) *OrganisationsService {
	return &OrganisationsService{organisationsRepo: repo}
}

func (s *OrganisationsService) CreateOrganisation(ctx context.Context) (*models.Organisation, error) {
	log.Printf("📋 Starting to create organization")

	organization := &models.Organisation{
		ID: core.NewID("org"),
	}

	if err := s.organisationsRepo.CreateOrganisation(ctx, organization); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	log.Printf("📋 Completed successfully - created organization with ID: %s", organization.ID)
	return organization, nil
}

func (s *OrganisationsService) GetOrganisationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.Organisation], error) {
	log.Printf("📋 Starting to get organization by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.Organisation](), fmt.Errorf("organization ID must be a valid ULID")
	}

	organization, err := s.organisationsRepo.GetOrganisationByID(ctx, id)
	if err != nil {
		return mo.None[*models.Organisation](), fmt.Errorf("failed to get organization by ID: %w", err)
	}

	if organization.IsPresent() {
		log.Printf("📋 Completed successfully - retrieved organization with ID: %s", id)
	} else {
		log.Printf("📋 Completed successfully - organization not found with ID: %s", id)
	}
	return organization, nil
}
