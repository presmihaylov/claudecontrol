package settings

import (
	"context"

	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) UpsertBooleanSetting(
	ctx context.Context,
	organizationID string,
	key string,
	value bool,
) error {
	args := m.Called(ctx, organizationID, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) UpsertStringSetting(
	ctx context.Context,
	organizationID string,
	key string,
	value string,
) error {
	args := m.Called(ctx, organizationID, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) UpsertStringArraySetting(
	ctx context.Context,
	organizationID string,
	key string,
	value []string,
) error {
	args := m.Called(ctx, organizationID, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) GetBooleanSetting(
	ctx context.Context,
	organizationID string,
	key string,
) (bool, error) {
	args := m.Called(ctx, organizationID, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockSettingsService) GetStringSetting(
	ctx context.Context,
	organizationID string,
	key string,
) (string, error) {
	args := m.Called(ctx, organizationID, key)
	return args.String(0), args.Error(1)
}

func (m *MockSettingsService) GetStringArraySetting(
	ctx context.Context,
	organizationID string,
	key string,
) ([]string, error) {
	args := m.Called(ctx, organizationID, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSettingsService) GetSettingByType(
	ctx context.Context,
	organizationID string,
	key string,
	settingType models.SettingType,
) (any, error) {
	args := m.Called(ctx, organizationID, key, settingType)
	return args.Get(0), args.Error(1)
}
