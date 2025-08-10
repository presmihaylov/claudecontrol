package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/appctx"
	"ccbackend/models"
	"ccbackend/models/api"
	discordintegrations "ccbackend/services/discord_integrations"
	organizations "ccbackend/services/organizations"
	slackintegrations "ccbackend/services/slack_integrations"
	users "ccbackend/services/users"
)

// Test data
var (
	testOrg = &models.Organization{
		ID: "org_01234567890123456789012345",
	}

	testUser = &models.User{
		ID:             "u_01234567890123456789012345",
		AuthProvider:   "clerk",
		AuthProviderID: "user_test_123",
		OrganizationID: testOrg.ID,
	}

	testSlackIntegration = &models.SlackIntegration{
		ID:             "si_01234567890123456789012345",
		SlackTeamID:    "T123456",
		SlackAuthToken: "xoxb-test-token",
		SlackTeamName:  "Test Team",
		OrganizationID: testOrg.ID,
	}
)

// Helper function to create context with user and organization
func contextWithUser(user *models.User) context.Context {
	ctx := appctx.SetUser(context.Background(), user)
	ctx = appctx.SetOrganization(ctx, testOrg)
	return ctx
}

func TestDashboardAPIHandler_ListSlackIntegrations(t *testing.T) {
	tests := []struct {
		name           string
		user           *models.User
		mockSetup      func(*slackintegrations.MockSlackIntegrationsService)
		expectedResult []*models.SlackIntegration
		expectedError  string
	}{
		{
			name: "success - returns integrations",
			user: testUser,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testOrg.ID).
					Return([]*models.SlackIntegration{testSlackIntegration}, nil)
			},
			expectedResult: []*models.SlackIntegration{testSlackIntegration},
			expectedError:  "",
		},
		{
			name: "success - no integrations",
			user: testUser,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testOrg.ID).
					Return([]*models.SlackIntegration{}, nil)
			},
			expectedResult: []*models.SlackIntegration{},
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockSlackIntegrationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)

			result, err := handler.ListSlackIntegrations(context.Background(), tt.user)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

func TestDashboardAPIHandler_CreateSlackIntegration(t *testing.T) {
	tests := []struct {
		name           string
		slackAuthToken string
		redirectURL    string
		user           *models.User
		mockSetup      func(*slackintegrations.MockSlackIntegrationsService)
		expectedResult *models.SlackIntegration
		expectedError  string
	}{
		{
			name:           "success - creates integration",
			slackAuthToken: "test-auth-code",
			redirectURL:    "https://example.com/redirect",
			user:           testUser,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("CreateSlackIntegration", mock.Anything, testOrg.ID, "test-auth-code", "https://example.com/redirect").
					Return(testSlackIntegration, nil)
			},
			expectedResult: testSlackIntegration,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockSlackIntegrationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)

			result, err := handler.CreateSlackIntegration(
				context.Background(),
				tt.slackAuthToken,
				tt.redirectURL,
				tt.user,
			)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

func TestDashboardAPIHandler_DeleteSlackIntegration(t *testing.T) {
	ctx := contextWithUser(testUser)
	integrationID := "si_01234567890123456789012345"

	tests := []struct {
		name          string
		ctx           context.Context
		integrationID string
		mockSetup     func(*slackintegrations.MockSlackIntegrationsService)
		expectedError string
	}{
		{
			name:          "success - deletes integration",
			ctx:           ctx,
			integrationID: integrationID,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("DeleteSlackIntegration", ctx, testOrg.ID, integrationID).Return(nil)
			},
			expectedError: "",
		},
		{
			name:          "error - service fails",
			ctx:           ctx,
			integrationID: integrationID,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("DeleteSlackIntegration", ctx, testOrg.ID, integrationID).Return(fmt.Errorf("not found"))
			},
			expectedError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockSlackIntegrationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)

			err := handler.DeleteSlackIntegration(tt.ctx, tt.integrationID)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

func TestDashboardAPIHandler_GenerateCCAgentSecretKey(t *testing.T) {
	ctx := contextWithUser(testUser)
	expectedSecretKey := "new-secret-key-123"

	tests := []struct {
		name           string
		ctx            context.Context
		mockSetup      func(*organizations.MockOrganizationsService)
		expectedResult string
		expectedError  string
	}{
		{
			name: "success - generates key",
			ctx:  ctx,
			mockSetup: func(m *organizations.MockOrganizationsService) {
				m.On("GenerateCCAgentSecretKey", ctx, testOrg.ID).Return(expectedSecretKey, nil)
			},
			expectedResult: expectedSecretKey,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockOrganizationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)

			result, err := handler.GenerateCCAgentSecretKey(tt.ctx)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

// HTTP Handler Tests

func TestDashboardHTTPHandler_HandleUserAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		user           *models.User
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "success - returns user data",
			method:         "POST",
			user:           testUser,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"u_01234567890123456789012345","auth_provider":"clerk","auth_provider_id":"user_test_123"}`,
		},
		{
			name:           "error - wrong method",
			method:         "GET",
			user:           testUser,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)
			httpHandler := NewDashboardHTTPHandler(handler)

			req := httptest.NewRequest(tt.method, "/users/authenticate", nil)
			if tt.user != nil {
				req = req.WithContext(contextWithUser(tt.user))
			}
			rr := httptest.NewRecorder()

			httpHandler.HandleUserAuthenticate(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response api.UserModel
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
				assert.Equal(t, tt.user.ID, response.ID)
			} else {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

func TestDashboardHTTPHandler_HandleListSlackIntegrations(t *testing.T) {
	tests := []struct {
		name           string
		user           *models.User
		mockSetup      func(*slackintegrations.MockSlackIntegrationsService)
		expectedStatus int
		validateBody   func(*testing.T, []byte)
	}{
		{
			name: "success - returns integrations",
			user: testUser,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testOrg.ID).
					Return([]*models.SlackIntegration{testSlackIntegration}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response []api.SlackIntegrationModel
				require.NoError(t, json.Unmarshal(body, &response))
				assert.Len(t, response, 1)
				assert.Equal(t, testSlackIntegration.ID, response[0].ID)
				assert.Equal(t, testSlackIntegration.SlackTeamName, response[0].SlackTeamName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockSlackIntegrationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)
			httpHandler := NewDashboardHTTPHandler(handler)

			req := httptest.NewRequest("GET", "/slack/integrations", nil)
			req = req.WithContext(contextWithUser(tt.user))
			rr := httptest.NewRecorder()

			httpHandler.HandleListSlackIntegrations(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			tt.validateBody(t, rr.Body.Bytes())

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

func TestDashboardHTTPHandler_HandleCreateSlackIntegration(t *testing.T) {
	validRequest := SlackIntegrationRequest{
		SlackAuthToken: "test-auth-code",
		RedirectURL:    "https://example.com/redirect",
	}

	tests := []struct {
		name           string
		user           *models.User
		requestBody    any
		mockSetup      func(*slackintegrations.MockSlackIntegrationsService)
		expectedStatus int
		validateBody   func(*testing.T, []byte)
	}{
		{
			name:        "success - creates integration",
			user:        testUser,
			requestBody: validRequest,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("CreateSlackIntegration", mock.Anything, testOrg.ID, "test-auth-code", "https://example.com/redirect").
					Return(testSlackIntegration, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response api.SlackIntegrationModel
				require.NoError(t, json.Unmarshal(body, &response))
				assert.Equal(t, testSlackIntegration.ID, response.ID)
				assert.Equal(t, testSlackIntegration.SlackTeamName, response.SlackTeamName)
			},
		},
		{
			name: "error - missing token",
			user: testUser,
			requestBody: SlackIntegrationRequest{
				SlackAuthToken: "",
				RedirectURL:    "https://example.com/redirect",
			},
			mockSetup:      func(m *slackintegrations.MockSlackIntegrationsService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "slackAuthToken is required")
			},
		},
		{
			name:           "error - invalid json",
			user:           testUser,
			requestBody:    "invalid json",
			mockSetup:      func(m *slackintegrations.MockSlackIntegrationsService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "invalid request body")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockSlackIntegrationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)
			httpHandler := NewDashboardHTTPHandler(handler)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest("POST", "/slack/integrations", bytes.NewReader(body))
			req = req.WithContext(contextWithUser(tt.user))
			rr := httptest.NewRecorder()

			httpHandler.HandleCreateSlackIntegration(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			tt.validateBody(t, rr.Body.Bytes())

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

func TestDashboardHTTPHandler_HandleDeleteSlackIntegration(t *testing.T) {
	validID := "si_01234567890123456789012345"
	ctx := contextWithUser(testUser)

	tests := []struct {
		name           string
		integrationID  string
		mockSetup      func(*slackintegrations.MockSlackIntegrationsService)
		expectedStatus int
		validateBody   func(*testing.T, []byte)
	}{
		{
			name:          "success - deletes integration",
			integrationID: validID,
			mockSetup: func(m *slackintegrations.MockSlackIntegrationsService) {
				m.On("DeleteSlackIntegration", mock.AnythingOfType("*context.valueCtx"), testOrg.ID, validID).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			validateBody: func(t *testing.T, body []byte) {
				assert.Empty(t, string(body))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockSlackIntegrationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)
			httpHandler := NewDashboardHTTPHandler(handler)

			req := httptest.NewRequest("DELETE", fmt.Sprintf("/slack/integrations/%s", tt.integrationID), nil)
			req = req.WithContext(ctx)

			// Setup mux router to capture path variables
			router := mux.NewRouter()
			router.HandleFunc("/slack/integrations/{id}", httpHandler.HandleDeleteSlackIntegration)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			tt.validateBody(t, rr.Body.Bytes())

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}

func TestDashboardHTTPHandler_HandleGenerateCCAgentSecretKey(t *testing.T) {
	expectedSecretKey := "new-secret-key-123"
	ctx := contextWithUser(testUser)

	tests := []struct {
		name           string
		mockSetup      func(*organizations.MockOrganizationsService)
		expectedStatus int
		validateBody   func(*testing.T, []byte)
	}{
		{
			name: "success - generates key",
			mockSetup: func(m *organizations.MockOrganizationsService) {
				m.On("GenerateCCAgentSecretKey", mock.AnythingOfType("*context.valueCtx"), testOrg.ID).
					Return(expectedSecretKey, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response CCAgentSecretKeyResponse
				require.NoError(t, json.Unmarshal(body, &response))
				assert.Equal(t, expectedSecretKey, response.SecretKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsersService := &users.MockUsersService{}
			mockSlackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
			mockOrganizationsService := &organizations.MockOrganizationsService{}
			tt.mockSetup(mockOrganizationsService)

			mockDiscordIntegrationsService := &discordintegrations.MockDiscordIntegrationsService{}
			handler := NewDashboardAPIHandler(
				mockUsersService,
				mockSlackIntegrationsService,
				mockDiscordIntegrationsService,
				mockOrganizationsService,
			)
			httpHandler := NewDashboardHTTPHandler(handler)

			req := httptest.NewRequest(
				"POST",
				"/organizations/ccagent_secret_key",
				nil,
			)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			httpHandler.HandleGenerateCCAgentSecretKey(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			tt.validateBody(t, rr.Body.Bytes())

			mockUsersService.AssertExpectations(t)
			mockSlackIntegrationsService.AssertExpectations(t)
			mockOrganizationsService.AssertExpectations(t)
		})
	}
}
