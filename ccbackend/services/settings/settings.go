package settings

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/utils"
)

type SettingsService struct {
	settingsRepo *db.PostgresSettingsRepository
}

func NewSettingsService(repo *db.PostgresSettingsRepository) *SettingsService {
	return &SettingsService{settingsRepo: repo}
}

func (s *SettingsService) UpsertBooleanSetting(ctx context.Context, organizationID string, key string, value bool) error {
	log.Printf("📋 Starting to upsert boolean setting: %s", key)
	if err := s.validateKey(key, models.SettingTypeBool); err != nil {
		return fmt.Errorf("invalid setting: %w", err)
	}

	_, err := s.settingsRepo.UpsertBooleanSetting(
		ctx,
		organizationID,
		models.SettingScopeTypeOrg,
		"",
		key,
		value,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert boolean setting: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted boolean setting: %s", key)
	return nil
}

func (s *SettingsService) UpsertStringSetting(ctx context.Context, organizationID string, key string, value string) error {
	log.Printf("📋 Starting to upsert string setting: %s", key)
	if err := s.validateKey(key, models.SettingTypeString); err != nil {
		return err
	}

	_, err := s.settingsRepo.UpsertStringSetting(
		ctx,
		organizationID,
		models.SettingScopeTypeOrg,
		"",
		key,
		value,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert string setting: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted string setting: %s", key)
	return nil
}

func (s *SettingsService) UpsertStringArraySetting(ctx context.Context, organizationID string, key string, value []string) error {
	log.Printf("📋 Starting to upsert string array setting: %s", key)
	if err := s.validateKey(key, models.SettingTypeStringArr); err != nil {
		return fmt.Errorf("invalid setting: %w", err)
	}

	_, err := s.settingsRepo.UpsertStringArraySetting(
		ctx,
		organizationID,
		models.SettingScopeTypeOrg,
		"",
		key,
		value,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert string array setting: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted string array setting: %s", key)
	return nil
}

func (s *SettingsService) GetBooleanSetting(ctx context.Context, organizationID string, key string) (mo.Option[bool], error) {
	log.Printf("📋 Starting to get boolean setting: %s", key)
	if err := s.validateKey(key, models.SettingTypeBool); err != nil {
		return mo.None[bool](), fmt.Errorf("invalid setting: %w", err)
	}

	setting, err := s.settingsRepo.GetSetting(
		ctx,
		organizationID,
		models.SettingScopeTypeOrg,
		"",
		key,
	)
	if err != nil {
		if err.Error() == "setting not found" {
			log.Printf("📋 Completed successfully - boolean setting not found: %s", key)
			return mo.None[bool](), nil
		}

		return mo.None[bool](), fmt.Errorf("failed to get boolean setting: %w", err)
	}

	utils.AssertInvariant(setting.ValueBoolean != nil, "boolean setting must have a value")
	log.Printf("📋 Completed successfully - retrieved boolean setting: %s", key)
	return mo.Some(*setting.ValueBoolean), nil
}

func (s *SettingsService) GetStringSetting(ctx context.Context, organizationID string, key string) (mo.Option[string], error) {
	log.Printf("📋 Starting to get string setting: %s", key)
	if err := s.validateKey(key, models.SettingTypeString); err != nil {
		return mo.None[string](), err
	}

	setting, err := s.settingsRepo.GetSetting(
		ctx,
		organizationID,
		models.SettingScopeTypeOrg,
		"",
		key,
	)
	if err != nil {
		if err.Error() == "setting not found" {
			log.Printf("📋 Completed successfully - string setting not found: %s", key)
			return mo.None[string](), nil
		}

		return mo.None[string](), fmt.Errorf("failed to get string setting: %w", err)
	}

	utils.AssertInvariant(setting.ValueString != nil, "string setting must have a value")
	log.Printf("📋 Completed successfully - retrieved string setting: %s", key)
	return mo.Some(*setting.ValueString), nil
}

func (s *SettingsService) GetStringArraySetting(ctx context.Context, organizationID string, key string) (mo.Option[[]string], error) {
	log.Printf("📋 Starting to get string array setting: %s", key)
	if err := s.validateKey(key, models.SettingTypeStringArr); err != nil {
		return mo.None[[]string](), err
	}

	setting, err := s.settingsRepo.GetSetting(
		ctx,
		organizationID,
		models.SettingScopeTypeOrg,
		"",
		key,
	)
	if err != nil {
		if err.Error() == "setting not found" {
			log.Printf("📋 Completed successfully - string array setting not found: %s", key)
			return mo.None[[]string](), nil
		}
		return mo.None[[]string](), fmt.Errorf("failed to get string array setting: %w", err)
	}

	utils.AssertInvariant(setting.ValueStringArr != nil, "string array setting must have a value")
	log.Printf("📋 Completed successfully - retrieved string array setting: %s", key)
	return mo.Some([]string(setting.ValueStringArr)), nil
}

func (s *SettingsService) validateKey(key string, expectedType models.SettingType) error {
	keyDef, exists := models.SupportedSettings[key]
	if !exists {
		return fmt.Errorf("unsupported setting key: %s", key)
	}

	if keyDef.Type != expectedType {
		return fmt.Errorf("setting key %s expects type %s, got %s", key, keyDef.Type, expectedType)
	}

	return nil
}
