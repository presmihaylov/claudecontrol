package settings

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/appctx"
	"ccbackend/db"
	"ccbackend/models"
)

type SettingsService struct {
	settingsRepo *db.PostgresSettingsRepository
}

func NewSettingsService(repo *db.PostgresSettingsRepository) *SettingsService {
	return &SettingsService{settingsRepo: repo}
}

func (s *SettingsService) UpsertBooleanSetting(ctx context.Context, key string, value bool) error {
	log.Printf("📋 Starting to upsert boolean setting: %s", key)

	if err := s.validateKey(key, models.SettingTypeBool); err != nil {
		return err
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return fmt.Errorf("organization not found in context")
	}

	_, err := s.settingsRepo.UpsertBooleanSetting(
		ctx,
		org.ID,
		"org",
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

func (s *SettingsService) UpsertStringSetting(ctx context.Context, key string, value string) error {
	log.Printf("📋 Starting to upsert string setting: %s", key)

	if err := s.validateKey(key, models.SettingTypeString); err != nil {
		return err
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return fmt.Errorf("organization not found in context")
	}

	_, err := s.settingsRepo.UpsertStringSetting(
		ctx,
		org.ID,
		"org",
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

func (s *SettingsService) UpsertStringArraySetting(ctx context.Context, key string, value []string) error {
	log.Printf("📋 Starting to upsert string array setting: %s", key)

	if err := s.validateKey(key, models.SettingTypeStringArr); err != nil {
		return err
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return fmt.Errorf("organization not found in context")
	}

	_, err := s.settingsRepo.UpsertStringArraySetting(
		ctx,
		org.ID,
		"org",
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

func (s *SettingsService) GetBooleanSetting(ctx context.Context, key string) (mo.Option[bool], error) {
	log.Printf("📋 Starting to get boolean setting: %s", key)

	if err := s.validateKey(key, models.SettingTypeBool); err != nil {
		return mo.None[bool](), err
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return mo.None[bool](), fmt.Errorf("organization not found in context")
	}

	setting, err := s.settingsRepo.GetSetting(
		ctx,
		org.ID,
		"org",
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

	if setting.ValueBoolean == nil {
		log.Printf("📋 Completed successfully - boolean setting has no value: %s", key)
		return mo.None[bool](), nil
	}

	log.Printf("📋 Completed successfully - retrieved boolean setting: %s", key)
	return mo.Some(*setting.ValueBoolean), nil
}

func (s *SettingsService) GetStringSetting(ctx context.Context, key string) (mo.Option[string], error) {
	log.Printf("📋 Starting to get string setting: %s", key)

	if err := s.validateKey(key, models.SettingTypeString); err != nil {
		return mo.None[string](), err
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return mo.None[string](), fmt.Errorf("organization not found in context")
	}

	setting, err := s.settingsRepo.GetSetting(
		ctx,
		org.ID,
		"org",
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

	if setting.ValueString == nil {
		log.Printf("📋 Completed successfully - string setting has no value: %s", key)
		return mo.None[string](), nil
	}

	log.Printf("📋 Completed successfully - retrieved string setting: %s", key)
	return mo.Some(*setting.ValueString), nil
}

func (s *SettingsService) GetStringArraySetting(ctx context.Context, key string) (mo.Option[[]string], error) {
	log.Printf("📋 Starting to get string array setting: %s", key)

	if err := s.validateKey(key, models.SettingTypeStringArr); err != nil {
		return mo.None[[]string](), err
	}

	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return mo.None[[]string](), fmt.Errorf("organization not found in context")
	}

	setting, err := s.settingsRepo.GetSetting(
		ctx,
		org.ID,
		"org",
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

	if len(setting.ValueStringArr) == 0 {
		log.Printf("📋 Completed successfully - string array setting has no value: %s", key)
		return mo.None[[]string](), nil
	}

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
