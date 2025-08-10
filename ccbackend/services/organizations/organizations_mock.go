package organizations

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockOrganizationsService is a mock implementation of the OrganizationsService interface
type MockOrganizationsService struct {
	mock.Mock
}

func (m *MockOrganizationsService) CreateOrganization(ctx context.Context) (*models.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationsService) GetOrganizationByID(ctx context.Context, id string) (mo.Option[*models.Organization], error) {
	args := m.Called(ctx, id)
	return args.Get(0).(mo.Option[*models.Organization]), args.Error(1)
}

func (m *MockOrganizationsService) GetAllOrganizations(ctx context.Context) ([]*models.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrganizationsService) GenerateCCAgentSecretKey(ctx context.Context, organizationID string) (string, error) {
	args := m.Called(ctx, organizationID)
	return args.String(0), args.Error(1)
}

func (m *MockOrganizationsService) GetOrganizationBySecretKey(ctx context.Context, secretKey string) (mo.Option[*models.Organization], error) {
	args := m.Called(ctx, secretKey)
	return args.Get(0).(mo.Option[*models.Organization]), args.Error(1)
}