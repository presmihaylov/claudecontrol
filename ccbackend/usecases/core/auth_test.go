package core

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"

	"ccbackend/models"
)

func TestValidateAPIKey_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
	}

	apiKey := "test_api_key_123"
	expectedOrgID := "org_abc123"
	organization := &models.Organization{
		ID: expectedOrgID,
	}

	mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).Return(mo.Some(organization), nil)

	// Act
	orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedOrgID, orgID)
	mockOrganizationsService.AssertNumberOfCalls(t, "GetOrganizationBySecretKey", 1)
	mockOrganizationsService.AssertCalled(t, "GetOrganizationBySecretKey", ctx, apiKey)
}

func TestValidateAPIKey_InvalidKey(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
	}

	apiKey := "invalid_api_key"

	// Return empty Option (key not found)
	mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).Return(mo.None[*models.Organization](), nil)

	// Act
	orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
	assert.Empty(t, orgID)
	mockOrganizationsService.AssertNumberOfCalls(t, "GetOrganizationBySecretKey", 1)
	mockOrganizationsService.AssertCalled(t, "GetOrganizationBySecretKey", ctx, apiKey)
}

func TestValidateAPIKey_ServiceError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
	}

	apiKey := "test_api_key"
	serviceErr := errors.New("database connection error")

	mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).Return(mo.None[*models.Organization](), serviceErr)

	// Act
	orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, serviceErr, err)
	assert.Empty(t, orgID)
	mockOrganizationsService.AssertNumberOfCalls(t, "GetOrganizationBySecretKey", 1)
}

func TestValidateAPIKey_EmptyAPIKey(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockOrganizationsService := new(MockOrganizationsService)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
	}

	apiKey := ""

	// Service should be called even with empty string
	mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).Return(mo.None[*models.Organization](), nil)

	// Act
	orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
	assert.Empty(t, orgID)
	mockOrganizationsService.AssertNumberOfCalls(t, "GetOrganizationBySecretKey", 1)
	mockOrganizationsService.AssertCalled(t, "GetOrganizationBySecretKey", ctx, apiKey)
}

func TestValidateAPIKey_NilContext(t *testing.T) {
	// Setup
	mockOrganizationsService := new(MockOrganizationsService)
	useCase := &CoreUseCase{
		organizationsService: mockOrganizationsService,
	}

	apiKey := "test_api_key"
	organization := &models.Organization{
		ID: "org_123",
	}

	// nil context is handled by the underlying service layer
	// It should either handle gracefully or panic depending on implementation
	mockOrganizationsService.On("GetOrganizationBySecretKey", nil, apiKey).Return(mo.Some(organization), nil)

	// Act
	orgID, err := useCase.ValidateAPIKey(nil, apiKey)

	// Assert - assuming graceful handling
	assert.NoError(t, err)
	assert.Equal(t, "org_123", orgID)
	mockOrganizationsService.AssertNumberOfCalls(t, "GetOrganizationBySecretKey", 1)
}

func TestValidateAPIKey_OptionTypeHandling(t *testing.T) {
	testCases := []struct {
		name           string
		returnOption   mo.Option[*models.Organization]
		returnError    error
		expectedOrgID  string
		expectedError  string
	}{
		{
			name: "Present_Option_With_Valid_Org",
			returnOption: mo.Some(&models.Organization{
				ID: "org_valid",
			}),
			returnError:   nil,
			expectedOrgID: "org_valid",
			expectedError: "",
		},
		{
			name:          "Empty_Option_No_Error",
			returnOption:  mo.None[*models.Organization](),
			returnError:   nil,
			expectedOrgID: "",
			expectedError: "invalid API key",
		},
		{
			name:          "Empty_Option_With_Error",
			returnOption:  mo.None[*models.Organization](),
			returnError:   errors.New("service error"),
			expectedOrgID: "",
			expectedError: "service error",
		},
		{
			name: "Present_Option_With_Error",
			returnOption: mo.Some(&models.Organization{
				ID: "org_test",
			}),
			returnError:   errors.New("unexpected error"),
			expectedOrgID: "",
			expectedError: "unexpected error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			ctx := context.Background()
			mockOrganizationsService := new(MockOrganizationsService)
			useCase := &CoreUseCase{
				organizationsService: mockOrganizationsService,
			}

			apiKey := "test_key"
			mockOrganizationsService.On("GetOrganizationBySecretKey", ctx, apiKey).Return(tc.returnOption, tc.returnError)

			// Act
			orgID, err := useCase.ValidateAPIKey(ctx, apiKey)

			// Assert
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedOrgID, orgID)
		})
	}
}