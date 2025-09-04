package settings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/testutils"
)

func setupSettingsTest(t *testing.T) (*SettingsService, *models.Organization, context.Context, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)

	settingsRepo := db.NewPostgresSettingsRepository(dbConn, cfg.DatabaseSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	service := NewSettingsService(settingsRepo)

	// Create a test organization
	org := &models.Organization{
		ID: core.NewID("org"),
	}
	err = organizationsRepo.CreateOrganization(context.Background(), org)
	require.NoError(t, err)

	cleanup := func() {
		dbConn.Close()
	}

	return service, org, context.Background(), cleanup
}

func TestSettingsService_UpsertBooleanSetting(t *testing.T) {
	service, org, ctx, cleanup := setupSettingsTest(t)
	defer cleanup()

	t.Run("successful upsert of boolean setting", func(t *testing.T) {
		key := "org/onboarding_finished"
		value := true

		err := service.UpsertBooleanSetting(ctx, org.ID, key, value)
		assert.NoError(t, err)

		// Verify the setting was created
		retrievedValue, err := service.GetBooleanSetting(ctx, org.ID, key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrievedValue)
	})

	t.Run("upsert overwrites existing value", func(t *testing.T) {
		key := "org/onboarding_finished"

		// First upsert
		err := service.UpsertBooleanSetting(ctx, org.ID, key, false)
		require.NoError(t, err)

		// Second upsert with different value
		err = service.UpsertBooleanSetting(ctx, org.ID, key, true)
		require.NoError(t, err)

		// Verify the new value
		retrievedValue, err := service.GetBooleanSetting(ctx, org.ID, key)
		assert.NoError(t, err)
		assert.Equal(t, true, retrievedValue)
	})

	t.Run("fails with unsupported key", func(t *testing.T) {
		err := service.UpsertBooleanSetting(ctx, org.ID, "invalid/key", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported setting key")
	})

	t.Run("fails with wrong type for key", func(t *testing.T) {
		key := "org/onboarding_finished" // This is defined as bool type
		err := service.UpsertStringSetting(ctx, org.ID, key, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects type bool, got string")
	})

	t.Run("fails with invalid organization ID", func(t *testing.T) {
		err := service.UpsertBooleanSetting(context.Background(), "invalid_org_id", "org/onboarding_finished", true)
		assert.Error(t, err)
		// This will fail at database level when trying to insert setting with non-existent org ID
		assert.Contains(t, err.Error(), "failed to upsert boolean setting")
	})
}

func TestSettingsService_GetBooleanSetting(t *testing.T) {
	service, org, ctx, cleanup := setupSettingsTest(t)
	defer cleanup()

	t.Run("returns default value when setting does not exist", func(t *testing.T) {
		retrievedValue, err := service.GetBooleanSetting(ctx, org.ID, "org/onboarding_finished")
		assert.NoError(t, err)
		assert.Equal(t, false, retrievedValue) // Default value from SupportedSettings
	})

	t.Run("returns value when setting exists", func(t *testing.T) {
		key := "org/onboarding_finished"
		expectedValue := true

		// First create the setting
		err := service.UpsertBooleanSetting(ctx, org.ID, key, expectedValue)
		require.NoError(t, err)

		// Then retrieve it
		retrievedValue, err := service.GetBooleanSetting(ctx, org.ID, key)
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, retrievedValue)
	})

	t.Run("fails with unsupported key", func(t *testing.T) {
		_, err := service.GetBooleanSetting(ctx, org.ID, "invalid/key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported setting key")
	})

	t.Run("returns default value for invalid organization ID", func(t *testing.T) {
		retrievedValue, err := service.GetBooleanSetting(
			context.Background(),
			"invalid_org_id",
			"org/onboarding_finished",
		)
		assert.NoError(t, err)
		assert.Equal(t, false, retrievedValue) // Should return default value even when org doesn't exist
	})
}

func TestSettingsService_DefaultValues(t *testing.T) {
	service, org, ctx, cleanup := setupSettingsTest(t)
	defer cleanup()

	t.Run("boolean setting returns default value when not set", func(t *testing.T) {
		value, err := service.GetBooleanSetting(ctx, org.ID, "org/onboarding_finished")
		assert.NoError(t, err)
		assert.Equal(t, false, value) // Default value from models
	})

	t.Run("boolean setting returns stored value over default", func(t *testing.T) {
		key := "org/onboarding_finished"

		// Set a value different from default (default is false)
		err := service.UpsertBooleanSetting(ctx, org.ID, key, true)
		require.NoError(t, err)

		value, err := service.GetBooleanSetting(ctx, org.ID, key)
		assert.NoError(t, err)
		assert.Equal(t, true, value) // Stored value, not default
	})
}

func TestSettingsService_GetSettingByType_DefaultValues(t *testing.T) {
	service, org, ctx, cleanup := setupSettingsTest(t)
	defer cleanup()

	t.Run("returns default boolean value when not set", func(t *testing.T) {
		value, err := service.GetSettingByType(ctx, org.ID, "org/onboarding_finished", models.SettingTypeBool)
		assert.NoError(t, err)
		assert.Equal(t, false, value)
	})

	t.Run("returns stored value over default", func(t *testing.T) {
		key := "org/onboarding_finished"

		// Set a value different from default
		err := service.UpsertBooleanSetting(ctx, org.ID, key, true)
		require.NoError(t, err)

		value, err := service.GetSettingByType(ctx, org.ID, key, models.SettingTypeBool)
		assert.NoError(t, err)
		assert.Equal(t, true, value) // Stored value, not default
	})
}

func TestSettingsService_ValidateKey(t *testing.T) {
	service := &SettingsService{}

	t.Run("validates correct key and type", func(t *testing.T) {
		err := service.validateKey("org/onboarding_finished", models.SettingTypeBool)
		assert.NoError(t, err)
	})

	t.Run("rejects unsupported key", func(t *testing.T) {
		err := service.validateKey("invalid/key", models.SettingTypeBool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported setting key")
	})

	t.Run("rejects wrong type for key", func(t *testing.T) {
		err := service.validateKey("org/onboarding_finished", models.SettingTypeString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects type bool, got string")
	})
}
