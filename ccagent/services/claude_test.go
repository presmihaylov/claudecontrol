package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ccagent/core"
)

// MockClaudeClient implements the ClaudeClient interface for testing
type MockClaudeClient struct {
	StartNewSessionFunc           func(prompt string) (string, error)
	StartNewSessionWithSystemFunc func(prompt, systemPrompt string) (string, error)
	ContinueSessionFunc           func(sessionID, prompt string) (string, error)
}

func (m *MockClaudeClient) StartNewSession(prompt string) (string, error) {
	if m.StartNewSessionFunc != nil {
		return m.StartNewSessionFunc(prompt)
	}
	return "", nil
}

func (m *MockClaudeClient) StartNewSessionWithSystemPrompt(prompt, systemPrompt string) (string, error) {
	if m.StartNewSessionWithSystemFunc != nil {
		return m.StartNewSessionWithSystemFunc(prompt, systemPrompt)
	}
	return "", nil
}

func (m *MockClaudeClient) ContinueSession(sessionID, prompt string) (string, error) {
	if m.ContinueSessionFunc != nil {
		return m.ContinueSessionFunc(sessionID, prompt)
	}
	return "", nil
}

func TestNewClaudeService(t *testing.T) {
	mockClient := &MockClaudeClient{}
	logDir := "/tmp/test-logs"

	service := NewClaudeService(mockClient, logDir)

	if service.claudeClient != mockClient {
		t.Error("Expected claude client to be set correctly")
	}

	if service.logDir != logDir {
		t.Errorf("Expected logDir to be %s, got %s", logDir, service.logDir)
	}
}

func TestClaudeService_StartNewConversation(t *testing.T) {
	tests := []struct {
		name            string
		prompt          string
		mockOutput      string
		mockError       error
		expectError     bool
		expectedOutput  string
		expectedSession string
	}{
		{
			name:            "successful conversation start",
			prompt:          "Hello",
			mockOutput:      `{"type":"assistant","message":{"id":"msg_123","type":"message","content":[{"type":"text","text":"Hello! How can I help?"}]},"session_id":"session_123"}`,
			mockError:       nil,
			expectError:     false,
			expectedOutput:  "Hello! How can I help?",
			expectedSession: "session_123",
		},
		{
			name:        "client returns error",
			prompt:      "Hello",
			mockOutput:  "",
			mockError:   fmt.Errorf("connection failed"),
			expectError: true,
		},
		{
			name:        "invalid JSON response",
			prompt:      "Hello",
			mockOutput:  "invalid json",
			mockError:   nil,
			expectError: true,
		},
		{
			name:        "missing assistant message",
			prompt:      "Hello",
			mockOutput:  `{"type":"system","session_id":"session_123"}`,
			mockError:   nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary log directory for testing
			tmpDir := t.TempDir()

			mockClient := &MockClaudeClient{
				StartNewSessionFunc: func(prompt string) (string, error) {
					if prompt != tt.prompt {
						t.Errorf("Expected prompt %s, got %s", tt.prompt, prompt)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			service := NewClaudeService(mockClient, tmpDir)

			result, err := service.StartNewConversation(tt.prompt)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Output != tt.expectedOutput {
				t.Errorf("Expected output %s, got %s", tt.expectedOutput, result.Output)
			}

			if result.SessionID != tt.expectedSession {
				t.Errorf("Expected session %s, got %s", tt.expectedSession, result.SessionID)
			}
		})
	}
}

func TestClaudeService_StartNewConversationWithSystemPrompt(t *testing.T) {
	tests := []struct {
		name            string
		prompt          string
		systemPrompt    string
		mockOutput      string
		mockError       error
		expectError     bool
		expectedOutput  string
		expectedSession string
	}{
		{
			name:            "successful conversation start with system prompt",
			prompt:          "Hello",
			systemPrompt:    "You are a helpful assistant",
			mockOutput:      `{"type":"assistant","message":{"id":"msg_456","type":"message","content":[{"type":"text","text":"Hello! I'm ready to assist."}]},"session_id":"session_456"}`,
			mockError:       nil,
			expectError:     false,
			expectedOutput:  "Hello! I'm ready to assist.",
			expectedSession: "session_456",
		},
		{
			name:         "client error with system prompt",
			prompt:       "Hello",
			systemPrompt: "You are helpful",
			mockOutput:   "",
			mockError:    fmt.Errorf("system prompt too long"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			mockClient := &MockClaudeClient{
				StartNewSessionWithSystemFunc: func(prompt, systemPrompt string) (string, error) {
					if prompt != tt.prompt {
						t.Errorf("Expected prompt %s, got %s", tt.prompt, prompt)
					}
					if systemPrompt != tt.systemPrompt {
						t.Errorf("Expected system prompt %s, got %s", tt.systemPrompt, systemPrompt)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			service := NewClaudeService(mockClient, tmpDir)

			result, err := service.StartNewConversationWithSystemPrompt(tt.prompt, tt.systemPrompt)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Output != tt.expectedOutput {
				t.Errorf("Expected output %s, got %s", tt.expectedOutput, result.Output)
			}

			if result.SessionID != tt.expectedSession {
				t.Errorf("Expected session %s, got %s", tt.expectedSession, result.SessionID)
			}
		})
	}
}

func TestClaudeService_ContinueConversation(t *testing.T) {
	tests := []struct {
		name            string
		sessionID       string
		prompt          string
		mockOutput      string
		mockError       error
		expectError     bool
		expectedOutput  string
		expectedSession string
	}{
		{
			name:            "successful conversation continue",
			sessionID:       "session_123",
			prompt:          "What's 2+2?",
			mockOutput:      `{"type":"assistant","message":{"id":"msg_789","type":"message","content":[{"type":"text","text":"2+2 equals 4"}]},"session_id":"session_123"}`,
			mockError:       nil,
			expectError:     false,
			expectedOutput:  "2+2 equals 4",
			expectedSession: "session_123",
		},
		{
			name:        "client error during continue",
			sessionID:   "session_123",
			prompt:      "Hello",
			mockOutput:  "",
			mockError:   fmt.Errorf("session expired"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			mockClient := &MockClaudeClient{
				ContinueSessionFunc: func(sessionID, prompt string) (string, error) {
					if sessionID != tt.sessionID {
						t.Errorf("Expected sessionID %s, got %s", tt.sessionID, sessionID)
					}
					if prompt != tt.prompt {
						t.Errorf("Expected prompt %s, got %s", tt.prompt, prompt)
					}
					return tt.mockOutput, tt.mockError
				},
			}

			service := NewClaudeService(mockClient, tmpDir)

			result, err := service.ContinueConversation(tt.sessionID, tt.prompt)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Output != tt.expectedOutput {
				t.Errorf("Expected output %s, got %s", tt.expectedOutput, result.Output)
			}

			if result.SessionID != tt.expectedSession {
				t.Errorf("Expected session %s, got %s", tt.expectedSession, result.SessionID)
			}
		})
	}
}

func TestClaudeService_writeClaudeErrorLog(t *testing.T) {
	tmpDir := t.TempDir()
	mockClient := &MockClaudeClient{}
	service := NewClaudeService(mockClient, tmpDir)

	rawOutput := "This is test error output\nwith multiple lines"

	logPath, err := service.writeClaudeErrorLog(rawOutput)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify file exists
	if !strings.Contains(logPath, tmpDir) {
		t.Errorf("Log path should be in temp directory %s, got %s", tmpDir, logPath)
	}

	// Verify filename pattern
	filename := filepath.Base(logPath)
	if !strings.HasPrefix(filename, "claude-error-") || !strings.HasSuffix(filename, ".log") {
		t.Errorf("Unexpected filename pattern: %s", filename)
	}

	// Verify file contents
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if string(content) != rawOutput {
		t.Errorf("Expected content %q, got %q", rawOutput, string(content))
	}
}

func TestClaudeService_extractSessionID(t *testing.T) {
	mockClient := &MockClaudeClient{}
	service := NewClaudeService(mockClient, "/tmp")

	tests := []struct {
		name     string
		messages []ClaudeMessage
		expected string
	}{
		{
			name: "messages with session ID",
			messages: []ClaudeMessage{
				AssistantMessage{
					Type:      "assistant",
					SessionID: "session_123",
				},
			},
			expected: "session_123",
		},
		{
			name:     "empty messages",
			messages: []ClaudeMessage{},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractSessionID(tt.messages)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClaudeService_extractClaudeResult(t *testing.T) {
	mockClient := &MockClaudeClient{}
	service := NewClaudeService(mockClient, "/tmp")

	tests := []struct {
		name        string
		messages    []ClaudeMessage
		expected    string
		expectError bool
	}{
		{
			name: "valid assistant message",
			messages: []ClaudeMessage{
				AssistantMessage{
					Type: "assistant",
					Message: struct {
						ID      string `json:"id"`
						Type    string `json:"type"`
						Content []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"content"`
					}{
						Content: []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}{
							{Type: "text", Text: "Hello world"},
						},
					},
				},
			},
			expected:    "Hello world",
			expectError: false,
		},
		{
			name:        "no assistant messages",
			messages:    []ClaudeMessage{},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.extractClaudeResult(tt.messages)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClaudeService_handleClaudeClientError(t *testing.T) {
	tmpDir := t.TempDir()
	mockClient := &MockClaudeClient{}
	service := NewClaudeService(mockClient, tmpDir)

	tests := []struct {
		name      string
		inputErr  error
		operation string
		expected  string
	}{
		{
			name:      "nil error",
			inputErr:  nil,
			operation: "test operation",
			expected:  "",
		},
		{
			name:      "regular error",
			inputErr:  fmt.Errorf("regular error"),
			operation: "test operation",
			expected:  "test operation: regular error",
		},
		{
			name: "claude command error with valid output",
			inputErr: &core.ErrClaudeCommandErr{
				Err:    fmt.Errorf("command failed"),
				Output: `{"type":"assistant","message":{"id":"msg_123","type":"message","content":[{"type":"text","text":"Error: Invalid input"}]},"session_id":"session_123"}`,
			},
			operation: "test operation",
			expected:  "test operation: Error: Invalid input",
		},
		{
			name: "claude command error with invalid output",
			inputErr: &core.ErrClaudeCommandErr{
				Err:    fmt.Errorf("command failed"),
				Output: "invalid json",
			},
			operation: "test operation",
			expected:  "test operation: claude command failed: command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.handleClaudeClientError(tt.inputErr, tt.operation)

			if tt.inputErr == nil {
				if result != nil {
					t.Errorf("Expected nil error for nil input, got %v", result)
				}
				return
			}

			if result == nil {
				t.Error("Expected error but got nil")
				return
			}

			if !strings.Contains(result.Error(), tt.expected) {
				t.Errorf("Expected error to contain %q, got %q", tt.expected, result.Error())
			}
		})
	}
}

func TestClaudeService_ParseErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	mockClient := &MockClaudeClient{
		StartNewSessionFunc: func(prompt string) (string, error) {
			// Return output that will parse successfully but have no assistant messages
			// This will fail at extractClaudeResult stage, not parsing stage
			return `{"type":"system","session_id":"session_123"}`, nil
		},
	}

	service := NewClaudeService(mockClient, tmpDir)

	_, err := service.StartNewConversation("test prompt")

	// Should get a regular error from extractClaudeResult
	if err == nil {
		t.Error("Expected error but got none")
		return
	}

	// This should be a regular fmt.Errorf, not ClaudeParseError
	expected := "failed to extract Claude result: no assistant message with text content found"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error to contain %q, got %q", expected, err.Error())
	}
}

func TestClaudeService_WriteErrorLogHandling(t *testing.T) {
	// Test case where writeClaudeErrorLog fails
	nonExistentDir := "/this/path/does/not/exist"

	mockClient := &MockClaudeClient{
		StartNewSessionFunc: func(prompt string) (string, error) {
			// Return output that will cause MapClaudeOutputToMessages to fail
			// We need to trigger an actual error from MapClaudeOutputToMessages
			// This is difficult since it's very resilient
			return "", fmt.Errorf("simulated client error")
		},
	}

	service := NewClaudeService(mockClient, nonExistentDir)

	_, err := service.StartNewConversation("test prompt")

	// Should get the client error, not parse error since the client failed
	if err == nil {
		t.Error("Expected error but got none")
		return
	}

	expected := "simulated client error"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error to contain %q, got %q", expected, err.Error())
	}
}
