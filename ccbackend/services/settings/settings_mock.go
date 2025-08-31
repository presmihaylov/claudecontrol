package settings

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"
)

type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) UpsertBooleanSetting(ctx context.Context, organizationID string, key string, value bool) error {
	args := m.Called(ctx, organizationID, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) UpsertStringSetting(ctx context.Context, organizationID string, key string, value string) error {
	args := m.Called(ctx, organizationID, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) UpsertStringArraySetting(ctx context.Context, organizationID string, key string, value []string) error {
	args := m.Called(ctx, organizationID, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) GetBooleanSetting(ctx context.Context, organizationID string, key string) (mo.Option[bool], error) {
	args := m.Called(ctx, organizationID, key)
	if args.Get(0) == nil {
		return mo.None[bool](), args.Error(1)
	}
	return args.Get(0).(mo.Option[bool]), args.Error(1)
}

func (m *MockSettingsService) GetStringSetting(ctx context.Context, organizationID string, key string) (mo.Option[string], error) {
	args := m.Called(ctx, organizationID, key)
	if args.Get(0) == nil {
		return mo.None[string](), args.Error(1)
	}
	return args.Get(0).(mo.Option[string]), args.Error(1)
}

func (m *MockSettingsService) GetStringArraySetting(ctx context.Context, organizationID string, key string) (mo.Option[[]string], error) {
	args := m.Called(ctx, organizationID, key)
	if args.Get(0) == nil {
		return mo.None[[]string](), args.Error(1)
	}
	return args.Get(0).(mo.Option[[]string]), args.Error(1)
}
