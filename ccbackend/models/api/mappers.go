package api

import "ccbackend/models"

// DomainUserToAPIUser converts a domain User model to an API UserModel
func DomainUserToAPIUser(domainUser *models.User) *UserModel {
	if domainUser == nil {
		return nil
	}

	return &UserModel{
		ID:        domainUser.ID,
		Email:     domainUser.Email,
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
		ID:            domainIntegration.ID,
		SlackTeamID:   domainIntegration.SlackTeamID,
		SlackTeamName: domainIntegration.SlackTeamName,
		OrgID:         domainIntegration.OrgID,
		CreatedAt:     domainIntegration.CreatedAt,
		UpdatedAt:     domainIntegration.UpdatedAt,
	}
}

// DomainSlackIntegrationsToAPISlackIntegrations converts a slice of domain SlackIntegration models to API SlackIntegrationModel slice
func DomainSlackIntegrationsToAPISlackIntegrations(
	domainIntegrations []models.SlackIntegration,
) []*SlackIntegrationModel {
	if domainIntegrations == nil {
		return nil
	}

	apiIntegrations := make([]*SlackIntegrationModel, len(domainIntegrations))
	for i, domainIntegration := range domainIntegrations {
		apiIntegrations[i] = DomainSlackIntegrationToAPISlackIntegration(&domainIntegration)
	}

	return apiIntegrations
}

// DomainDiscordIntegrationToAPIDiscordIntegration converts a domain DiscordIntegration model to an API DiscordIntegrationModel
func DomainDiscordIntegrationToAPIDiscordIntegration(
	domainIntegration *models.DiscordIntegration,
) *DiscordIntegrationModel {
	if domainIntegration == nil {
		return nil
	}

	return &DiscordIntegrationModel{
		ID:               domainIntegration.ID,
		DiscordGuildID:   domainIntegration.DiscordGuildID,
		DiscordGuildName: domainIntegration.DiscordGuildName,
		OrgID:            domainIntegration.OrgID,
		CreatedAt:        domainIntegration.CreatedAt,
		UpdatedAt:        domainIntegration.UpdatedAt,
	}
}

// DomainDiscordIntegrationsToAPIDiscordIntegrations converts a slice of domain DiscordIntegration models to API DiscordIntegrationModel slice
func DomainDiscordIntegrationsToAPIDiscordIntegrations(
	domainIntegrations []models.DiscordIntegration,
) []*DiscordIntegrationModel {
	if domainIntegrations == nil {
		return nil
	}

	apiIntegrations := make([]*DiscordIntegrationModel, len(domainIntegrations))
	for i, domainIntegration := range domainIntegrations {
		apiIntegrations[i] = DomainDiscordIntegrationToAPIDiscordIntegration(&domainIntegration)
	}

	return apiIntegrations
}
