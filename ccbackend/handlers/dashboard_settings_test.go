package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/models"
	agents "ccbackend/services/agents"
	anthropicintegrations "ccbackend/services/anthropic_integrations"
	ccagentcontainerintegrations "ccbackend/services/ccagent_container_integrations"
	discordintegrations "ccbackend/services/discord_integrations"
	githubintegrations "ccbackend/services/github_integrations"
	organizations "ccbackend/services/organizations"
	settingsservice "ccbackend/services/settings"
	slackintegrations "ccbackend/services/slack_integrations"
	users "ccbackend/services/users"
)

// Test data for settings
var (
	testBooleanSetting = true
	testStringSetting  = "test-value"
	testStringArraySetting = []string{"value1", "value2", "value3"}
)

func TestDashboardAPIHandler_UpsertSetting(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		settingType models.SettingType
		value       any
		mockSetup   func(*settingsservice.MockSettingsService)
	}{
		{
			name:        "success - boolean setting",
			key:         "org/onboarding_finished",
			settingType: models.SettingTypeBool,
			value:       testBooleanSetting,
			mockSetup: func(m *settingsservice.MockSettingsService) {
				m.On("UpsertBooleanSetting", mock.Anything, testOrg.ID, "org/onboarding_finished", testBooleanSetting).
					Return(nil)
			},
		},
		{
			name:        "success - string setting",
			key:         "org/test_string",
			settingType: models.SettingTypeString,
			value:       testStringSetting,
			mockSetup: func(m *settingsservice.MockSettingsService) {
				m.On("UpsertStringSetting", mock.Anything, testOrg.ID, "org/test_string", testStringSetting).
					Return(nil)
			},
		},
		{
			name:        "success - string array setting",
			key:         "org/test_array",
			settingType: models.SettingTypeStringArr,
			value:       testStringArraySetting,
			mockSetup: func(m *settingsservice.MockSettingsService) {
				m.On("UpsertStringArraySetting", mock.Anything, testOrg.ID, "org/test_array", testStringArraySetting).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			mockGitHubIntegrationsService := &githubintegrations.MockGitHubIntegrationsService{}
			mockAnthropicIntegrationsService := &anthropicintegrations.MockAnthropicIntegrationsService{}
			mockCCAgentContainerIntegrationsService := &ccagentcontainerintegrations.MockCCAgentContainerIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			mockAgentsService := &agents.MockAgentsService{}
			mockSettingsService := &settingsservice.MockSettingsService{}
			mockTxManager := &simpleTxManager{}

			tt.mockSetup(mockSettingsService)

			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockGitHubIntegrationsService,
				mockAnthropicIntegrationsService,
				mockCCAgentContainerIntegrationsService,
				mockOrganizationsService,
				mockAgentsService,
				mockSettingsService,
				mockTxManager,
			)

			ctx := contextWithUser(testUser)
			err := handler.UpsertSetting(ctx, tt.key, tt.settingType, tt.value)

			require.NoError(t, err)
			mockSettingsService.AssertExpectations(t)
		})
	}
}

func TestDashboardAPIHandler_GetSetting(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		expectedValue any
		expectedType  models.SettingType
		mockSetup     func(*settingsservice.MockSettingsService)
	}{
		{
			name:          "success - boolean setting",
			key:           "org/onboarding_finished",
			expectedValue: testBooleanSetting,
			expectedType:  models.SettingTypeBool,
			mockSetup: func(m *settingsservice.MockSettingsService) {
				m.On("GetSettingByType", mock.Anything, testOrg.ID, "org/onboarding_finished", models.SettingTypeBool).
					Return(testBooleanSetting, nil)
			},
		},
		{
			name:          "success - string setting",
			key:           "org/test_string",
			expectedValue: testStringSetting,
			expectedType:  models.SettingTypeString,
			mockSetup: func(m *settingsservice.MockSettingsService) {
				m.On("GetSettingByType", mock.Anything, testOrg.ID, "org/test_string", models.SettingTypeString).
					Return(testStringSetting, nil)
			},
		},
		{
			name:          "success - string array setting",
			key:           "org/test_array",
			expectedValue: testStringArraySetting,
			expectedType:  models.SettingTypeStringArr,
			mockSetup: func(m *settingsservice.MockSettingsService) {
				m.On("GetSettingByType", mock.Anything, testOrg.ID, "org/test_array", models.SettingTypeStringArr).
					Return(testStringArraySetting, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			mockGitHubIntegrationsService := &githubintegrations.MockGitHubIntegrationsService{}
			mockAnthropicIntegrationsService := &anthropicintegrations.MockAnthropicIntegrationsService{}
			mockCCAgentContainerIntegrationsService := &ccagentcontainerintegrations.MockCCAgentContainerIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			mockAgentsService := &agents.MockAgentsService{}
			mockSettingsService := &settingsservice.MockSettingsService{}
			mockTxManager := &simpleTxManager{}

			// Add the setting key to SupportedSettings for the test
			originalSupportedSettings := models.SupportedSettings
			models.SupportedSettings[tt.key] = models.SettingKeyDefinition{
				Key:  tt.key,
				Type: tt.expectedType,
			}
			defer func() {
				models.SupportedSettings = originalSupportedSettings
			}()

			tt.mockSetup(mockSettingsService)

			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockGitHubIntegrationsService,
				mockAnthropicIntegrationsService,
				mockCCAgentContainerIntegrationsService,
				mockOrganizationsService,
				mockAgentsService,
				mockSettingsService,
				mockTxManager,
			)

			ctx := contextWithUser(testUser)
			value, settingType, err := handler.GetSetting(ctx, tt.key)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedType, settingType)
			mockSettingsService.AssertExpectations(t)
		})
	}
}