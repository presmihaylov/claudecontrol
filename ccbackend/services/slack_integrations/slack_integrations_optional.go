package slackintegrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// OptionalSlackIntegrationsService returns errors for all operations when Slack is not configured
type OptionalSlackIntegrationsService struct{}

// NewOptionalSlackIntegrationsService creates a new optional Slack integrations service
func NewOptionalSlackIntegrationsService() *OptionalSlackIntegrationsService {
	return &OptionalSlackIntegrationsService{}
}

func (s *OptionalSlackIntegrationsService) CreateSlackIntegration(
	ctx context.Context,
	orgID models.OrgID,
	slackAuthCode, redirectURL string,
) (*models.SlackIntegration, error) {
	return nil, fmt.Errorf("Service Slack is not configured")
}

func (s *OptionalSlackIntegrationsService) GetSlackIntegrationsByOrganizationID(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.SlackIntegration, error) {
	return nil, fmt.Errorf("Service Slack is not configured")
}

func (s *OptionalSlackIntegrationsService) GetAllSlackIntegrations(ctx context.Context) ([]models.SlackIntegration, error) {
	return nil, fmt.Errorf("Service Slack is not configured")
}

func (s *OptionalSlackIntegrationsService) DeleteSlackIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error {
	return fmt.Errorf("Service Slack is not configured")
}

func (s *OptionalSlackIntegrationsService) GetSlackIntegrationByTeamID(ctx context.Context, teamID string) (mo.Option[*models.SlackIntegration], error) {
	return mo.None[*models.SlackIntegration](), fmt.Errorf("Service Slack is not configured")
}

func (s *OptionalSlackIntegrationsService) GetSlackIntegrationByID(ctx context.Context, id string) (mo.Option[*models.SlackIntegration], error) {
	return mo.None[*models.SlackIntegration](), fmt.Errorf("Service Slack is not configured")
}