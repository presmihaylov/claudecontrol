package commands

import (
	"context"

	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

type MockCommandsService struct {
	mock.Mock
}

func (m *MockCommandsService) ProcessCommand(
	ctx context.Context,
	request models.CommandRequest,
) (*models.CommandResult, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CommandResult), args.Error(1)
}