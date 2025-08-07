package api

import "ccbackend/models"

// DomainUserToAPIUser converts a domain User model to an API UserModel
func DomainUserToAPIUser(domainUser *models.User) *UserModel {
	if domainUser == nil {
		return nil
	}

	return &UserModel{
		ID:        domainUser.ID,
		CreatedAt: domainUser.CreatedAt,
		UpdatedAt: domainUser.UpdatedAt,
	}
}

// DomainSlackIntegrationToAPISlackIntegration converts a domain SlackIntegration model to an API SlackIntegrationModel
func DomainSlackIntegrationToAPISlackIntegration(domainIntegration *models.SlackIntegration) *SlackIntegrationModel {
	if domainIntegration == nil {
		return nil
	}

	return &SlackIntegrationModel{
		ID:                          domainIntegration.ID,
		SlackTeamID:                 domainIntegration.SlackTeamID,
		SlackTeamName:               domainIntegration.SlackTeamName,
		UserID:                      domainIntegration.UserID,
		CCAgentSecretKeyGeneratedAt: domainIntegration.CCAgentSecretKeyGeneratedAt,
		CreatedAt:                   domainIntegration.CreatedAt,
		UpdatedAt:                   domainIntegration.UpdatedAt,
	}
}

// DomainSlackIntegrationsToAPISlackIntegrations converts a slice of domain SlackIntegration models to API SlackIntegrationModel slice
func DomainSlackIntegrationsToAPISlackIntegrations(
	domainIntegrations []*models.SlackIntegration,
) []*SlackIntegrationModel {
	if domainIntegrations == nil {
		return nil
	}

	apiIntegrations := make([]*SlackIntegrationModel, len(domainIntegrations))
	for i, domainIntegration := range domainIntegrations {
		apiIntegrations[i] = DomainSlackIntegrationToAPISlackIntegration(domainIntegration)
	}

	return apiIntegrations
}
