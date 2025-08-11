package users

import (
	"context"

	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockUsersService is a mock implementation of the UsersService interface
type MockUsersService struct {
	mock.Mock
}

func (m *MockUsersService) GetOrCreateUser(
	ctx context.Context,
	authProvider, authProviderID, email string,
) (*models.User, error) {
	args := m.Called(ctx, authProvider, authProviderID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
