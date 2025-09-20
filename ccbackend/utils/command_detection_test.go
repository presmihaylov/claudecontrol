package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectCommand(t *testing.T) {
	tests := []struct {
		name            string
		messageText     string
		expectedIsCmd   bool
		expectedCmdText string
	}{
		{
			name:            "Simple command with mention",
			messageText:     "<@U123456> --cmd repo=github.com/user/repo",
			expectedIsCmd:   true,
			expectedCmdText: "--cmd repo=github.com/user/repo",
		},
		{
			name:            "Command without mention",
			messageText:     "--cmd repo=github.com/user/repo",
			expectedIsCmd:   true,
			expectedCmdText: "--cmd repo=github.com/user/repo",
		},
		{
			name:            "Discord mention format",
			messageText:     "<@!123456789> --cmd repo=github.com/user/repo",
			expectedIsCmd:   true,
			expectedCmdText: "--cmd repo=github.com/user/repo",
		},
		{
			name:            "Command with extra whitespace",
			messageText:     "<@U123456>   --cmd repo=github.com/user/repo  ",
			expectedIsCmd:   true,
			expectedCmdText: "--cmd repo=github.com/user/repo",
		},
		{
			name:            "Not a command",
			messageText:     "<@U123456> Hello, can you help me?",
			expectedIsCmd:   false,
			expectedCmdText: "",
		},
		{
			name:            "Command in middle of text",
			messageText:     "<@U123456> Please run --cmd repo=github.com/user/repo for me",
			expectedIsCmd:   false,
			expectedCmdText: "",
		},
		{
			name:            "Empty message",
			messageText:     "",
			expectedIsCmd:   false,
			expectedCmdText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectCommand(tt.messageText)
			assert.Equal(t, tt.expectedIsCmd, result.IsCommand, "IsCommand mismatch")
			assert.Equal(t, tt.expectedCmdText, result.CommandText, "CommandText mismatch")
		})
	}
}

func TestStripMentions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Slack mention simple",
			input:    "<@U123456> hello world",
			expected: "hello world",
		},
		{
			name:     "Slack mention with username",
			input:    "<@U123456|username> hello world",
			expected: "hello world",
		},
		{
			name:     "Discord mention simple",
			input:    "<@123456789> hello world",
			expected: "hello world",
		},
		{
			name:     "Discord mention with exclamation",
			input:    "<@!123456789> hello world",
			expected: "hello world",
		},
		{
			name:     "Multiple mentions",
			input:    "<@U123456> <@U789012> hello world",
			expected: "hello world",
		},
		{
			name:     "Mixed Slack and Discord mentions",
			input:    "<@U123456> <@!789012345> hello world",
			expected: "hello world",
		},
		{
			name:     "No mentions",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "Only mention",
			input:    "<@U123456>",
			expected: "",
		},
		{
			name:     "Mention at end",
			input:    "hello world <@U123456>",
			expected: "hello world",
		},
		{
			name:     "Multiple spaces after mention",
			input:    "<@U123456>    hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripMentions(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}