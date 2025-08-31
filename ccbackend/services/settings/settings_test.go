package settings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/appctx"
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

	// Create context with organization
	ctx := appctx.SetOrganization(context.Background(), org)

	cleanup := func() {
		dbConn.Close()
	}

	return service, org, ctx, cleanup
}

func TestSettingsService_UpsertBooleanSetting(t *testing.T) {
	service, _, ctx, cleanup := setupSettingsTest(t)
	defer cleanup()

	t.Run("successful upsert of boolean setting", func(t *testing.T) {
		key := "org/onboarding_finished"
		value := true

		err := service.UpsertBooleanSetting(ctx, key, value)
		assert.NoError(t, err)

		// Verify the setting was created
		valueOpt, err := service.GetBooleanSetting(ctx, key)
		assert.NoError(t, err)
		retrievedValue, ok := valueOpt.Get()
		assert.True(t, ok)
		assert.Equal(t, value, retrievedValue)
	})

	t.Run("upsert overwrites existing value", func(t *testing.T) {
		key := "org/onboarding_finished"

		// First upsert
		err := service.UpsertBooleanSetting(ctx, key, false)
		require.NoError(t, err)

		// Second upsert with different value
		err = service.UpsertBooleanSetting(ctx, key, true)
		require.NoError(t, err)

		// Verify the new value
		valueOpt, err := service.GetBooleanSetting(ctx, key)
		assert.NoError(t, err)
		retrievedValue, ok := valueOpt.Get()
		assert.True(t, ok)
		assert.Equal(t, true, retrievedValue)
	})

	t.Run("fails with unsupported key", func(t *testing.T) {
		err := service.UpsertBooleanSetting(ctx, "invalid/key", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported setting key")
	})

	t.Run("fails with wrong type for key", func(t *testing.T) {
		key := "org/onboarding_finished" // This is defined as bool type
		err := service.UpsertStringSetting(ctx, key, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expects type bool, got string")
	})

	t.Run("fails without organization in context", func(t *testing.T) {
		err := service.UpsertBooleanSetting(context.Background(), "org/onboarding_finished", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization not found in context")
	})
}

func TestSettingsService_GetBooleanSetting(t *testing.T) {
	service, _, ctx, cleanup := setupSettingsTest(t)
	defer cleanup()

	t.Run("returns none when setting does not exist", func(t *testing.T) {
		valueOpt, err := service.GetBooleanSetting(ctx, "org/onboarding_finished")
		assert.NoError(t, err)
		_, ok := valueOpt.Get()
		assert.False(t, ok)
	})

	t.Run("returns value when setting exists", func(t *testing.T) {
		key := "org/onboarding_finished"
		expectedValue := true

		// First create the setting
		err := service.UpsertBooleanSetting(ctx, key, expectedValue)
		require.NoError(t, err)

		// Then retrieve it
		valueOpt, err := service.GetBooleanSetting(ctx, key)
		assert.NoError(t, err)
		retrievedValue, ok := valueOpt.Get()
		assert.True(t, ok)
		assert.Equal(t, expectedValue, retrievedValue)
	})

	t.Run("fails with unsupported key", func(t *testing.T) {
		_, err := service.GetBooleanSetting(ctx, "invalid/key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported setting key")
	})

	t.Run("fails without organization in context", func(t *testing.T) {
		_, err := service.GetBooleanSetting(context.Background(), "org/onboarding_finished")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "organization not found in context")
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
