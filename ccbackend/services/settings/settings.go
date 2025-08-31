package settings

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/core"
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

func (s *SettingsService) UpsertBooleanSetting(
	ctx context.Context,
	organizationID string,
	key string,
	value bool,
) error {
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

func (s *SettingsService) UpsertStringSetting(
	ctx context.Context,
	organizationID string,
	key string,
	value string,
) error {
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

func (s *SettingsService) UpsertStringArraySetting(
	ctx context.Context,
	organizationID string,
	key string,
	value []string,
) error {
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

func (s *SettingsService) GetBooleanSetting(
	ctx context.Context,
	organizationID string,
	key string,
) (mo.Option[bool], error) {
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
		if core.IsNotFoundError(err) {
			defaultValue := s.getDefaultValue(key, models.SettingTypeBool)
			log.Printf("📋 Completed successfully - boolean setting not found, returning default: %s", key)
			return mo.Some(defaultValue.(bool)), nil
		}

		return mo.None[bool](), fmt.Errorf("failed to get boolean setting: %w", err)
	}

	utils.AssertInvariant(setting.ValueBoolean != nil, "boolean setting must have a value")
	log.Printf("📋 Completed successfully - retrieved boolean setting: %s", key)
	return mo.Some(*setting.ValueBoolean), nil
}

func (s *SettingsService) GetStringSetting(
	ctx context.Context,
	organizationID string,
	key string,
) (mo.Option[string], error) {
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
		if core.IsNotFoundError(err) {
			defaultValue := s.getDefaultValue(key, models.SettingTypeString)
			log.Printf("📋 Completed successfully - string setting not found, returning default: %s", key)
			return mo.Some(defaultValue.(string)), nil
		}

		return mo.None[string](), fmt.Errorf("failed to get string setting: %w", err)
	}

	utils.AssertInvariant(setting.ValueString != nil, "string setting must have a value")
	log.Printf("📋 Completed successfully - retrieved string setting: %s", key)
	return mo.Some(*setting.ValueString), nil
}

func (s *SettingsService) GetStringArraySetting(
	ctx context.Context,
	organizationID string,
	key string,
) (mo.Option[[]string], error) {
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
		if core.IsNotFoundError(err) {
			defaultValue := s.getDefaultValue(key, models.SettingTypeStringArr)
			log.Printf("📋 Completed successfully - string array setting not found, returning default: %s", key)
			return mo.Some(defaultValue.([]string)), nil
		}
		return mo.None[[]string](), fmt.Errorf("failed to get string array setting: %w", err)
	}

	utils.AssertInvariant(setting.ValueStringArr != nil, "string array setting must have a value")
	log.Printf("📋 Completed successfully - retrieved string array setting: %s", key)
	return mo.Some([]string(setting.ValueStringArr)), nil
}

func (s *SettingsService) GetSettingByType(
	ctx context.Context,
	organizationID string,
	key string,
	settingType models.SettingType,
) (any, error) {
	log.Printf("📋 Starting to get setting by type: %s (type: %s)", key, settingType)

	switch settingType {
	case models.SettingTypeBool:
		valueOpt, err := s.GetBooleanSetting(ctx, organizationID, key)
		if err != nil {
			return nil, err
		}
		value, ok := valueOpt.Get()
		utils.AssertInvariant(
			ok,
			fmt.Sprintf("boolean setting %s must always have a value (either stored or default)", key),
		)
		log.Printf("📋 Completed successfully - retrieved boolean setting: %s", key)
		return value, nil
	case models.SettingTypeString:
		valueOpt, err := s.GetStringSetting(ctx, organizationID, key)
		if err != nil {
			return nil, err
		}
		value, ok := valueOpt.Get()
		utils.AssertInvariant(
			ok,
			fmt.Sprintf("string setting %s must always have a value (either stored or default)", key),
		)
		log.Printf("📋 Completed successfully - retrieved string setting: %s", key)
		return value, nil
	case models.SettingTypeStringArr:
		valueOpt, err := s.GetStringArraySetting(ctx, organizationID, key)
		if err != nil {
			return nil, err
		}
		value, ok := valueOpt.Get()
		utils.AssertInvariant(
			ok,
			fmt.Sprintf("string array setting %s must always have a value (either stored or default)", key),
		)
		log.Printf("📋 Completed successfully - retrieved string array setting: %s", key)
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported setting type: %s", settingType)
	}
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

func (s *SettingsService) getDefaultValue(key string, expectedType models.SettingType) any {
	keyDef, exists := models.SupportedSettings[key]
	if !exists || keyDef.Type != expectedType {
		return nil
	}

	utils.AssertInvariant(keyDef.DefaultValue != nil, fmt.Sprintf("setting %s must have a default value", key))

	switch expectedType {
	case models.SettingTypeBool:
		boolVal, ok := keyDef.DefaultValue.(bool)
		utils.AssertInvariant(ok, fmt.Sprintf("default value for boolean setting %s must be a bool", key))
		return boolVal
	case models.SettingTypeString:
		strVal, ok := keyDef.DefaultValue.(string)
		utils.AssertInvariant(ok, fmt.Sprintf("default value for string setting %s must be a string", key))
		return strVal
	case models.SettingTypeStringArr:
		strArrVal, ok := keyDef.DefaultValue.([]string)
		utils.AssertInvariant(ok, fmt.Sprintf("default value for string array setting %s must be a []string", key))
		return strArrVal
	default:
		utils.AssertInvariant(false, fmt.Sprintf("unsupported setting type: %s", expectedType))
		return nil
	}
}
