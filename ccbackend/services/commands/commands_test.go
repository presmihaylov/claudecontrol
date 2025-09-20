package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
	agentsservice "ccbackend/services/agents"
	connectedchannelsservice "ccbackend/services/connectedchannels"
)

func TestCommandsService_ProcessCommand_RepoCommand_Success(t *testing.T) {
	// Setup
	mockAgentsService := &agentsservice.MockAgentsService{}
	mockConnectedChannelsService := &connectedchannelsservice.MockConnectedChannelsService{}

	service := NewCommandsService(mockAgentsService, mockConnectedChannelsService)

	orgID := models.OrgID("o_test123")
	normalizedRepoURL := "github.com/user/repo"

	// Mock connected channel
	connectedChannel := &models.SlackConnectedChannel{
		ID:             "channel1",
		OrgID:          orgID,
		TeamID:         "team123",
		ChannelID:      "channel123",
		DefaultRepoURL: nil,
	}

	// Mock agents with matching repository
	agents := []*models.ActiveAgent{
		{
			ID:      "agent1",
			RepoURL: normalizedRepoURL,
		},
	}

	// Expected Slack channel update
	updatedSlackChannel := &models.SlackConnectedChannel{
		ID:             "channel1",
		OrgID:          orgID,
		TeamID:         "team123",
		ChannelID:      "channel123",
		DefaultRepoURL: &normalizedRepoURL,
	}

	// Setup mocks
	mockAgentsService.On("GetAvailableAgents", mock.Anything, orgID).Return(agents, nil)
	mockConnectedChannelsService.On("UpdateSlackChannelDefaultRepo", mock.Anything, orgID, "team123", "channel123", normalizedRepoURL).Return(updatedSlackChannel, nil)

	request := models.CommandRequest{
		Command:     "--cmd repo=github.com/user/repo",
		UserID:      "user123",
		MessageText: "--cmd repo=github.com/user/repo",
	}

	// Execute
	result, err := service.ProcessCommand(context.Background(), request, connectedChannel)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "✅ Repository set to github.com/user/repo")

	// Verify mocks
	mockAgentsService.AssertExpectations(t)
	mockConnectedChannelsService.AssertExpectations(t)
}

func TestCommandsService_ProcessCommand_RepoCommand_RepositoryNotFound(t *testing.T) {
	// Setup
	mockAgentsService := &agentsservice.MockAgentsService{}
	mockConnectedChannelsService := &connectedchannelsservice.MockConnectedChannelsService{}

	service := NewCommandsService(mockAgentsService, mockConnectedChannelsService)

	orgID := models.OrgID("o_test123")

	// Mock connected channel
	connectedChannel := &models.SlackConnectedChannel{
		ID:             "channel1",
		OrgID:          orgID,
		TeamID:         "team123",
		ChannelID:      "channel123",
		DefaultRepoURL: nil,
	}

	// Mock agents with different repository
	agents := []*models.ActiveAgent{
		{
			ID:      "agent1",
			RepoURL: "github.com/other/repo",
		},
	}

	// Setup mocks
	mockAgentsService.On("GetAvailableAgents", mock.Anything, orgID).Return(agents, nil)

	request := models.CommandRequest{
		Command:     "--cmd repo=github.com/user/repo",
		UserID:      "user123",
		MessageText: "--cmd repo=github.com/user/repo",
	}

	// Execute
	result, err := service.ProcessCommand(context.Background(), request, connectedChannel)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "Repository github.com/user/repo not found in active agents")

	// Verify mocks
	mockAgentsService.AssertExpectations(t)
	mockConnectedChannelsService.AssertExpectations(t)
}

func TestCommandsService_ProcessCommand_DiscordCommand_Success(t *testing.T) {
	// Setup
	mockAgentsService := &agentsservice.MockAgentsService{}
	mockConnectedChannelsService := &connectedchannelsservice.MockConnectedChannelsService{}

	service := NewCommandsService(mockAgentsService, mockConnectedChannelsService)

	orgID := models.OrgID("o_test123")
	normalizedRepoURL := "github.com/user/repo"

	// Mock connected channel (Discord)
	connectedChannel := &models.DiscordConnectedChannel{
		ID:             "channel1",
		OrgID:          orgID,
		GuildID:        "guild123",
		ChannelID:      "channel123",
		DefaultRepoURL: nil,
	}

	// Mock agents with matching repository
	agents := []*models.ActiveAgent{
		{
			ID:      "agent1",
			RepoURL: normalizedRepoURL,
		},
	}

	// Expected Discord channel update
	updatedDiscordChannel := &models.DiscordConnectedChannel{
		ID:             "channel1",
		OrgID:          orgID,
		GuildID:        "guild123",
		ChannelID:      "channel123",
		DefaultRepoURL: &normalizedRepoURL,
	}

	// Setup mocks
	mockAgentsService.On("GetAvailableAgents", mock.Anything, orgID).Return(agents, nil)
	mockConnectedChannelsService.On("UpdateDiscordChannelDefaultRepo", mock.Anything, orgID, "guild123", "channel123", normalizedRepoURL).Return(updatedDiscordChannel, nil)

	request := models.CommandRequest{
		Command:     "--cmd repo=github.com/user/repo",
		UserID:      "user123",
		MessageText: "--cmd repo=github.com/user/repo",
	}

	// Execute
	result, err := service.ProcessCommand(context.Background(), request, connectedChannel)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "✅ Repository set to github.com/user/repo")

	// Verify mocks
	mockAgentsService.AssertExpectations(t)
	mockConnectedChannelsService.AssertExpectations(t)
}

func TestCommandsService_ProcessCommand_InvalidCommand(t *testing.T) {
	// Setup
	mockAgentsService := &agentsservice.MockAgentsService{}
	mockConnectedChannelsService := &connectedchannelsservice.MockConnectedChannelsService{}

	service := NewCommandsService(mockAgentsService, mockConnectedChannelsService)

	orgID := models.OrgID("o_test123")

	// Mock connected channel
	connectedChannel := &models.SlackConnectedChannel{
		ID:             "channel1",
		OrgID:          orgID,
		TeamID:         "team123",
		ChannelID:      "channel123",
		DefaultRepoURL: nil,
	}

	request := models.CommandRequest{
		Command:     "--cmd invalid=value",
		UserID:      "user123",
		MessageText: "--cmd invalid=value",
	}

	// Execute
	result, err := service.ProcessCommand(context.Background(), request, connectedChannel)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "Unknown command: invalid")

	// Verify mocks (no expectations set, so they should not be called)
	mockAgentsService.AssertExpectations(t)
	mockConnectedChannelsService.AssertExpectations(t)
}

func TestCommandsService_ParseCommand(t *testing.T) {
	service := &CommandsService{}

	tests := []struct {
		name                string
		command             string
		expectedCommandType string
		expectedValue       string
		expectError         bool
	}{
		{
			name:                "Valid repo command",
			command:             "--cmd repo=github.com/user/repo",
			expectedCommandType: "repo",
			expectedValue:       "github.com/user/repo",
			expectError:         false,
		},
		{
			name:                "Valid repo command with spaces",
			command:             "  --cmd   repo=github.com/user/repo  ",
			expectedCommandType: "repo",
			expectedValue:       "github.com/user/repo",
			expectError:         false,
		},
		{
			name:        "Missing --cmd prefix",
			command:     "repo=github.com/user/repo",
			expectError: true,
		},
		{
			name:        "Missing equals sign",
			command:     "--cmd repo",
			expectError: true,
		},
		{
			name:        "Empty command value",
			command:     "--cmd repo=",
			expectError: true,
		},
		{
			name:        "Empty command type",
			command:     "--cmd =github.com/user/repo",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commandType, value, err := service.parseCommand(tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCommandType, commandType)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestCommandsService_NormalizeRepoURL(t *testing.T) {
	service := &CommandsService{}

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "Already normalized",
			input:    "github.com/user/repo",
			expected: "github.com/user/repo",
		},
		{
			name:     "HTTPS URL",
			input:    "https://github.com/user/repo",
			expected: "github.com/user/repo",
		},
		{
			name:     "HTTP URL",
			input:    "http://github.com/user/repo",
			expected: "github.com/user/repo",
		},
		{
			name:     "With trailing slash",
			input:    "https://github.com/user/repo/",
			expected: "github.com/user/repo",
		},
		{
			name:     "Slack link format",
			input:    "<https://github.com/user/repo>",
			expected: "github.com/user/repo",
		},
		{
			name:        "Invalid URL",
			input:       "not-a-github-url",
			expectError: true,
		},
		{
			name:        "Non-GitHub domain",
			input:       "https://gitlab.com/user/repo",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.normalizeRepoURL(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}