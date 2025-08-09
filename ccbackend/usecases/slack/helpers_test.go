package slack

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/models"
)

func TestSendStartConversationToAgent(t *testing.T) {
	t.Run("Success_SendToAgent", func(t *testing.T) {
		// Setup
		useCase, _, _, _, _, mockSocketClient, _ := setupSlackUseCase(t)
		
		// Mock expectations
		mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Execute
		err := useCase.sendStartConversationToAgent(
			context.Background(),
			testAgent.WSConnectionID,
			testProcessedMessage,
		)
		
		// Assert
		require.NoError(t, err)
		mockSocketClient.AssertExpectations(t)
	})
}

func TestSendUserMessageToAgent(t *testing.T) {
	t.Run("Success_SendToAgent", func(t *testing.T) {
		// Setup
		useCase, _, _, _, _, mockSocketClient, _ := setupSlackUseCase(t)
		
		// Mock expectations
		mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Execute
		err := useCase.sendUserMessageToAgent(
			context.Background(),
			testAgent.WSConnectionID,
			testProcessedMessage,
		)
		
		// Assert
		require.NoError(t, err)
		mockSocketClient.AssertExpectations(t)
	})
}