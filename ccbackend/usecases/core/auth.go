package core

import (
	"context"
	"fmt"
	"log"
)

// ValidateAPIKey validates an API key and returns the organization ID if valid
func (s *CoreUseCase) ValidateAPIKey(ctx context.Context, apiKey string) (string, error) {
	log.Printf("📋 Starting to validate API key")

	maybeOrg, err := s.organizationsService.GetOrganizationBySecretKey(ctx, apiKey)
	if err != nil {
		return "", err
	}
	if !maybeOrg.IsPresent() {
		return "", fmt.Errorf("invalid API key")
	}
	organization := maybeOrg.MustGet()

	log.Printf("📋 Completed successfully - validated API key for organization %s", organization.ID)
	return organization.ID, nil
}
