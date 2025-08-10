package discord

import (
	"ccbackend/models"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimDiscordMessage(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedLength int
		expectTrimmed  bool
	}{
		{
			name:           "Short message - no trimming needed",
			input:          "Hello, this is a short message",
			expectedLength: 30,
			expectTrimmed:  false,
		},
		{
			name:           "Exactly 2000 characters - no trimming needed",
			input:          strings.Repeat("a", 2000),
			expectedLength: 2000,
			expectTrimmed:  false,
		},
		{
			name:           "2001 characters - should be trimmed",
			input:          strings.Repeat("a", 2001),
			expectedLength: 2000,
			expectTrimmed:  true,
		},
		{
			name:           "Long message - should be trimmed with ellipsis",
			input:          strings.Repeat("This is a long message. ", 100), // ~2400 characters
			expectedLength: 2000,
			expectTrimmed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimDiscordMessage(tt.input)

			// Check the result length is within Discord's limit
			assert.LessOrEqual(t, len(result), 2000, "Result should not exceed Discord's 2000 character limit")

			// Check expected length
			assert.Equal(t, tt.expectedLength, len(result), "Result length should match expected")

			if tt.expectTrimmed {
				// Should be shorter than input and end with ellipsis
				assert.Less(t, len(result), len(tt.input), "Trimmed message should be shorter than input")
				assert.True(t, strings.HasSuffix(result, "..."), "Trimmed message should end with ellipsis")
			} else {
				// Should be identical to input
				assert.Equal(t, tt.input, result, "Short message should not be modified")
			}
		})
	}
}

func TestTrimDiscordMessageEdgeCases(t *testing.T) {
	t.Run("Empty message", func(t *testing.T) {
		result := trimDiscordMessage("")
		assert.Equal(t, "", result, "Empty message should remain empty")
	})

	t.Run("Message with exactly 1997 characters leaves room for ellipsis", func(t *testing.T) {
		input := strings.Repeat("a", 2005) // 5 chars over limit
		result := trimDiscordMessage(input)

		assert.Equal(t, 2000, len(result), "Result should be exactly 2000 characters")
		assert.True(t, strings.HasSuffix(result, "..."), "Should end with ellipsis")
		assert.Equal(t, 1997, len(strings.TrimSuffix(result, "...")), "Should have 1997 characters before ellipsis")
	})
}

func TestDeriveMessageReactionFromStatus(t *testing.T) {
	t.Run("in_progress_status", func(t *testing.T) {
		result := deriveMessageReactionFromStatus(models.ProcessedDiscordMessageStatusInProgress)
		assert.Equal(t, EmojiHourglass, result)
	})

	t.Run("queued_status", func(t *testing.T) {
		result := deriveMessageReactionFromStatus(models.ProcessedDiscordMessageStatusQueued)
		assert.Equal(t, EmojiHourglass, result)
	})

	t.Run("completed_status", func(t *testing.T) {
		result := deriveMessageReactionFromStatus(models.ProcessedDiscordMessageStatusCompleted)
		assert.Equal(t, EmojiCheckMark, result)
	})
}

func TestIsAgentErrorMessage(t *testing.T) {
	t.Run("is_agent_error", func(t *testing.T) {
		result := isAgentErrorMessage("ccagent encountered error: something went wrong")
		assert.True(t, result)
	})

	t.Run("not_agent_error", func(t *testing.T) {
		result := isAgentErrorMessage("regular system message")
		assert.False(t, result)
	})
}
