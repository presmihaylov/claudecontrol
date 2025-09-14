package slackintegrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// UnconfiguredSlackIntegrationsService returns errors for all operations when Slack is not configured
type UnconfiguredSlackIntegrationsService struct{}

// NewUnconfiguredSlackIntegrationsService creates a new unconfigured Slack integrations service
func NewUnconfiguredSlackIntegrationsService() *UnconfiguredSlackIntegrationsService {
	return &UnconfiguredSlackIntegrationsService{}
}

func (s *UnconfiguredSlackIntegrationsService) CreateSlackIntegration(
	ctx context.Context,
	orgID models.OrgID,
	slackAuthCode, redirectURL string,
) (*models.SlackIntegration, error) {
	return nil, fmt.Errorf("service Slack is not configured")
}

func (s *UnconfiguredSlackIntegrationsService) GetSlackIntegrationsByOrganizationID(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.SlackIntegration, error) {
	return nil, fmt.Errorf("service Slack is not configured")
}

func (s *UnconfiguredSlackIntegrationsService) GetAllSlackIntegrations(
	ctx context.Context,
) ([]models.SlackIntegration, error) {
	return nil, fmt.Errorf("service Slack is not configured")
}

func (s *UnconfiguredSlackIntegrationsService) DeleteSlackIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	return fmt.Errorf("service Slack is not configured")
}

func (s *UnconfiguredSlackIntegrationsService) GetSlackIntegrationByTeamID(
	ctx context.Context,
	teamID string,
) (mo.Option[*models.SlackIntegration], error) {
	return mo.None[*models.SlackIntegration](), fmt.Errorf("service Slack is not configured")
}

func (s *UnconfiguredSlackIntegrationsService) GetSlackIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.SlackIntegration], error) {
	return mo.None[*models.SlackIntegration](), fmt.Errorf("service Slack is not configured")
}
