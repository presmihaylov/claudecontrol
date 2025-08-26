package api

import (
	"time"

	"ccbackend/models"
)

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

// DomainSlackIntegrationToAPISlackIntegration converts a domain SlackIntegration model to an API SlackIntegration
func DomainSlackIntegrationToAPISlackIntegration(domainIntegration *models.SlackIntegration) *SlackIntegration {
	if domainIntegration == nil {
		return nil
	}

	return &SlackIntegration{
		ID:            domainIntegration.ID,
		SlackTeamID:   domainIntegration.SlackTeamID,
		SlackTeamName: domainIntegration.SlackTeamName,
		OrgID:         domainIntegration.OrgID,
		CreatedAt:     domainIntegration.CreatedAt,
		UpdatedAt:     domainIntegration.UpdatedAt,
	}
}

// DomainSlackIntegrationsToAPISlackIntegrations converts a slice of domain SlackIntegration models to API SlackIntegration slice
func DomainSlackIntegrationsToAPISlackIntegrations(
	domainIntegrations []models.SlackIntegration,
) []*SlackIntegration {
	if domainIntegrations == nil {
		return nil
	}

	apiIntegrations := make([]*SlackIntegration, len(domainIntegrations))
	for i, domainIntegration := range domainIntegrations {
		apiIntegrations[i] = DomainSlackIntegrationToAPISlackIntegration(&domainIntegration)
	}

	return apiIntegrations
}

// DomainDiscordIntegrationToAPIDiscordIntegration converts a domain DiscordIntegration model to an API DiscordIntegration
func DomainDiscordIntegrationToAPIDiscordIntegration(
	domainIntegration *models.DiscordIntegration,
) *DiscordIntegration {
	if domainIntegration == nil {
		return nil
	}

	return &DiscordIntegration{
		ID:               domainIntegration.ID,
		DiscordGuildID:   domainIntegration.DiscordGuildID,
		DiscordGuildName: domainIntegration.DiscordGuildName,
		OrgID:            domainIntegration.OrgID,
		CreatedAt:        domainIntegration.CreatedAt,
		UpdatedAt:        domainIntegration.UpdatedAt,
	}
}

// DomainDiscordIntegrationsToAPIDiscordIntegrations converts a slice of domain DiscordIntegration models to API DiscordIntegration slice
func DomainDiscordIntegrationsToAPIDiscordIntegrations(
	domainIntegrations []models.DiscordIntegration,
) []*DiscordIntegration {
	if domainIntegrations == nil {
		return nil
	}

	apiIntegrations := make([]*DiscordIntegration, len(domainIntegrations))
	for i, domainIntegration := range domainIntegrations {
		apiIntegrations[i] = DomainDiscordIntegrationToAPIDiscordIntegration(&domainIntegration)
	}

	return apiIntegrations
}

// DomainGitHubIntegrationToAPIGitHubIntegration converts a domain GitHubIntegration model to an API GitHubIntegration
func DomainGitHubIntegrationToAPIGitHubIntegration(domainIntegration *models.GitHubIntegration) *GitHubIntegration {
	if domainIntegration == nil {
		return nil
	}

	return &GitHubIntegration{
		ID:                   domainIntegration.ID,
		GitHubInstallationID: domainIntegration.GitHubInstallationID,
		OrgID:                string(domainIntegration.OrgID),
		CreatedAt:            domainIntegration.CreatedAt,
		UpdatedAt:            domainIntegration.UpdatedAt,
	}
}

// DomainGitHubIntegrationsToAPIGitHubIntegrations converts a slice of domain GitHubIntegration models to API GitHubIntegration slice
func DomainGitHubIntegrationsToAPIGitHubIntegrations(
	domainIntegrations []models.GitHubIntegration,
) []*GitHubIntegration {
	if domainIntegrations == nil {
		return nil
	}

	apiIntegrations := make([]*GitHubIntegration, len(domainIntegrations))
	for i, domainIntegration := range domainIntegrations {
		apiIntegrations[i] = DomainGitHubIntegrationToAPIGitHubIntegration(&domainIntegration)
	}

	return apiIntegrations
}

// DomainAnthropicIntegrationToAPIAnthropicIntegration converts a domain AnthropicIntegration model to an API AnthropicIntegration
func DomainAnthropicIntegrationToAPIAnthropicIntegration(
	domainIntegration *models.AnthropicIntegration,
) *AnthropicIntegration {
	if domainIntegration == nil {
		return nil
	}

	api := &AnthropicIntegration{
		ID:            domainIntegration.ID,
		HasAPIKey:     domainIntegration.AnthropicAPIKey != nil,
		HasOAuthToken: domainIntegration.ClaudeCodeOAuthToken != nil,
		HasOAuthTokens: domainIntegration.ClaudeCodeAccessToken != nil &&
			domainIntegration.ClaudeCodeRefreshToken != nil,
		OrgID:     string(domainIntegration.OrgID),
		CreatedAt: domainIntegration.CreatedAt,
		UpdatedAt: domainIntegration.UpdatedAt,
	}

	// Check if OAuth tokens are expired
	if api.HasOAuthTokens && domainIntegration.AccessTokenExpiresAt != nil {
		api.OAuthTokenExpired = time.Now().After(*domainIntegration.AccessTokenExpiresAt)
	}

	return api
}

// DomainAnthropicIntegrationsToAPIAnthropicIntegrations converts a slice of domain AnthropicIntegration models to API AnthropicIntegration slice
func DomainAnthropicIntegrationsToAPIAnthropicIntegrations(
	domainIntegrations []models.AnthropicIntegration,
) []*AnthropicIntegration {
	if domainIntegrations == nil {
		return nil
	}

	apiIntegrations := make([]*AnthropicIntegration, len(domainIntegrations))
	for i, domainIntegration := range domainIntegrations {
		apiIntegrations[i] = DomainAnthropicIntegrationToAPIAnthropicIntegration(&domainIntegration)
	}

	return apiIntegrations
}
