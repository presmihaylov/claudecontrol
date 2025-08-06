package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ccagent/clients"
	"ccagent/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockClaudeClient implements the ClaudeClient interface for testing
type MockClaudeClient struct {
	startNewSessionOutput           string
	startNewSessionErr              error
	startNewSessionWithSystemOutput string
	startNewSessionWithSystemErr    error
	continueSessionOutput           string
	continueSessionErr              error
}

// Ensure MockClaudeClient implements the interface
var _ clients.ClaudeClient = (*MockClaudeClient)(nil)

func (m *MockClaudeClient) StartNewSession(prompt string) (string, error) {
	return m.startNewSessionOutput, m.startNewSessionErr
}

func (m *MockClaudeClient) StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error) {
	return m.startNewSessionWithSystemOutput, m.startNewSessionWithSystemErr
}

func (m *MockClaudeClient) ContinueSession(sessionID, prompt string) (string, error) {
	return m.continueSessionOutput, m.continueSessionErr
}

func TestNewClaudeService(t *testing.T) {
	mockClient := &MockClaudeClient{}
	logDir := "/tmp/test-logs"

	service := NewClaudeService(mockClient, logDir)

	assert.NotNil(t, service)
	assert.Equal(t, logDir, service.logDir)
}

func TestClaudeService_StartNewConversation_Success(t *testing.T) {
	mockClient := &MockClaudeClient{
		startNewSessionOutput: `{"type":"message","created_at":"2024-01-01T00:00:00Z","message":{"id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"Hello! How can I help you?"}],"model":"claude-3-sonnet-20240229","usage":{"input_tokens":10,"output_tokens":8}}}
{"type":"session","id":"session_456"}`,
	}

	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	result, err := service.StartNewConversation("Hello")

	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you?", result.Output)
	assert.Equal(t, "session_456", result.SessionID)
}

func TestClaudeService_StartNewConversationWithSystemPrompt_Success(t *testing.T) {
	mockClient := &MockClaudeClient{
		startNewSessionWithSystemOutput: `{"type":"message","created_at":"2024-01-01T00:00:00Z","message":{"id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"I understand the system prompt."}],"model":"claude-3-sonnet-20240229","usage":{"input_tokens":10,"output_tokens":8}}}
{"type":"session","id":"session_789"}`,
	}

	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	result, err := service.StartNewConversationWithSystemPrompt("Hello", "You are a helpful assistant")

	require.NoError(t, err)
	assert.Equal(t, "I understand the system prompt.", result.Output)
	assert.Equal(t, "session_789", result.SessionID)
}

func TestClaudeService_ContinueConversation_Success(t *testing.T) {
	mockClient := &MockClaudeClient{
		continueSessionOutput: `{"type":"message","created_at":"2024-01-01T00:00:00Z","message":{"id":"msg_124","type":"message","role":"assistant","content":[{"type":"text","text":"Continuing our conversation."}],"model":"claude-3-sonnet-20240229","usage":{"input_tokens":10,"output_tokens":8}}}
{"type":"session","id":"session_456"}`,
	}

	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	result, err := service.ContinueConversation("session_456", "Tell me more")

	require.NoError(t, err)
	assert.Equal(t, "Continuing our conversation.", result.Output)
	assert.Equal(t, "session_456", result.SessionID)
}

func TestClaudeService_StartNewConversation_ParseError(t *testing.T) {
	mockClient := &MockClaudeClient{
		startNewSessionOutput: "invalid json output",
	}

	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	result, err := service.StartNewConversation("Hello")

	require.Error(t, err)
	assert.Nil(t, result)

	// Check that error is a ClaudeParseError
	claudeParseErr, ok := err.(*core.ClaudeParseError)
	require.True(t, ok, "Expected ClaudeParseError, got %T", err)

	// Check that log file was created
	assert.True(t, strings.HasSuffix(claudeParseErr.LogFilePath, ".log"))
	assert.FileExists(t, claudeParseErr.LogFilePath)

	// Verify log file contains the raw output
	content, readErr := os.ReadFile(claudeParseErr.LogFilePath)
	require.NoError(t, readErr)
	assert.Equal(t, "invalid json output", string(content))

	// Clean up
	defer os.Remove(claudeParseErr.LogFilePath)
}

func TestClaudeService_StartNewConversation_ClientError(t *testing.T) {
	mockClient := &MockClaudeClient{
		startNewSessionErr: fmt.Errorf("connection failed"),
	}

	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	result, err := service.StartNewConversation("Hello")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to start new Claude session")
	assert.Contains(t, err.Error(), "connection failed")
}

func TestClaudeService_StartNewConversation_ClaudeCommandError(t *testing.T) {
	// Test handling of Claude command error with valid output
	claudeOutput := `{"type":"message","created_at":"2024-01-01T00:00:00Z","message":{"id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"I cannot help with that request."}],"model":"claude-3-sonnet-20240229","usage":{"input_tokens":10,"output_tokens":8}}}`

	mockClient := &MockClaudeClient{
		startNewSessionErr: &core.ErrClaudeCommandErr{
			Err:    fmt.Errorf("exit status 1"),
			Output: claudeOutput,
		},
	}

	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	result, err := service.StartNewConversation("Hello")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to start new Claude session")
	assert.Contains(t, err.Error(), "I cannot help with that request.")
}

func TestClaudeService_ContinueConversation_ClientError(t *testing.T) {
	mockClient := &MockClaudeClient{
		continueSessionErr: fmt.Errorf("session expired"),
	}

	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	result, err := service.ContinueConversation("session_456", "Continue")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to continue Claude session")
	assert.Contains(t, err.Error(), "session expired")
}

func TestClaudeService_WriteClaudeErrorLog(t *testing.T) {
	tempDir := t.TempDir()
	service := NewClaudeService(&MockClaudeClient{}, tempDir)

	testOutput := "This is test error output"

	logPath, err := service.writeClaudeErrorLog(testOutput)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(logPath, tempDir))
	assert.True(t, strings.Contains(filepath.Base(logPath), "claude-error-"))
	assert.True(t, strings.HasSuffix(logPath, ".log"))

	// Verify file contents
	content, readErr := os.ReadFile(logPath)
	require.NoError(t, readErr)
	assert.Equal(t, testOutput, string(content))

	// Verify file permissions
	info, statErr := os.Stat(logPath)
	require.NoError(t, statErr)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Clean up
	defer os.Remove(logPath)
}

func TestClaudeService_WriteClaudeErrorLog_InvalidDir(t *testing.T) {
	// Use a path that cannot be created (assuming /root is not writable)
	invalidDir := "/root/invalid/path"
	service := NewClaudeService(&MockClaudeClient{}, invalidDir)

	logPath, err := service.writeClaudeErrorLog("test")

	require.Error(t, err)
	assert.Empty(t, logPath)
	assert.Contains(t, err.Error(), "failed to create log directory")
}

func TestClaudeService_ExtractSessionID(t *testing.T) {
	mockClient := &MockClaudeClient{}
	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	// Test with valid messages
	assistantMsg := AssistantMessage{
		Type:      "message",
		SessionID: "session_123",
	}
	assistantMsg.Message.ID = "msg_456"
	assistantMsg.Message.Content = []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{{Type: "text", Text: "Hello"}}

	messages := []ClaudeMessage{assistantMsg}

	sessionID := service.extractSessionID(messages)
	assert.Equal(t, "session_123", sessionID)

	// Test with empty messages
	emptyMessages := []ClaudeMessage{}
	sessionID = service.extractSessionID(emptyMessages)
	assert.Equal(t, "unknown", sessionID)
}

func TestClaudeService_ExtractClaudeResult(t *testing.T) {
	mockClient := &MockClaudeClient{}
	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	// Test with valid assistant message
	assistantMsg := AssistantMessage{
		Type: "message",
	}
	assistantMsg.Message.ID = "msg_456"
	assistantMsg.Message.Content = []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{{Type: "text", Text: "Hello, how can I help?"}}

	messages := []ClaudeMessage{assistantMsg}

	result, err := service.extractClaudeResult(messages)
	require.NoError(t, err)
	assert.Equal(t, "Hello, how can I help?", result)

	// Test with no assistant message
	emptyMessages := []ClaudeMessage{}
	result, err = service.extractClaudeResult(emptyMessages)
	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no assistant message with text content found")

	// Test with assistant message but no text content
	assistantMsgNoText := AssistantMessage{
		Type: "message",
	}
	assistantMsgNoText.Message.ID = "msg_456"
	assistantMsgNoText.Message.Content = []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{{Type: "image", Text: ""}}

	messagesNoText := []ClaudeMessage{assistantMsgNoText}

	result, err = service.extractClaudeResult(messagesNoText)
	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no assistant message with text content found")
}

func TestClaudeService_HandleClaudeClientError(t *testing.T) {
	mockClient := &MockClaudeClient{}
	tempDir := t.TempDir()
	service := NewClaudeService(mockClient, tempDir)

	// Test with nil error
	result := service.handleClaudeClientError(nil, "test operation")
	assert.NoError(t, result)

	// Test with regular error (not Claude command error)
	regularErr := fmt.Errorf("regular error")
	result = service.handleClaudeClientError(regularErr, "test operation")
	require.Error(t, result)
	assert.Contains(t, result.Error(), "test operation")
	assert.Contains(t, result.Error(), "regular error")

	// Test with Claude command error containing valid assistant message
	claudeOutput := `{"type":"message","created_at":"2024-01-01T00:00:00Z","message":{"id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"Error message from Claude"}],"model":"claude-3-sonnet-20240229","usage":{"input_tokens":10,"output_tokens":8}}}`
	claudeErr := &core.ErrClaudeCommandErr{
		Err:    fmt.Errorf("exit status 1"),
		Output: claudeOutput,
	}

	result = service.handleClaudeClientError(claudeErr, "test operation")
	require.Error(t, result)
	assert.Contains(t, result.Error(), "test operation")
	assert.Contains(t, result.Error(), "Error message from Claude")

	// Test with Claude command error containing invalid JSON
	claudeErrInvalid := &core.ErrClaudeCommandErr{
		Err:    fmt.Errorf("exit status 1"),
		Output: "invalid json",
	}

	result = service.handleClaudeClientError(claudeErrInvalid, "test operation")
	require.Error(t, result)
	assert.Contains(t, result.Error(), "test operation")
}
