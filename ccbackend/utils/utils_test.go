package utils

import (
	"context"
	"errors"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConvertMarkdownToSlack(t *testing.T) {
	t.Run("ConvertBoldMarkdown", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "Single bold word",
				input:    "This is **bold** text",
				expected: "This is *bold* text",
			},
			{
				name:     "Multiple bold words",
				input:    "This is **bold** and this is **also bold**",
				expected: "This is *bold* and this is *also bold*",
			},
			{
				name:     "Bold phrase with spaces",
				input:    "This is **bold phrase** text",
				expected: "This is *bold phrase* text",
			},
			{
				name:     "No bold markdown",
				input:    "This is regular text",
				expected: "This is regular text",
			},
			{
				name:     "Empty string",
				input:    "",
				expected: "",
			},
			{
				name:     "Only bold text",
				input:    "**completely bold**",
				expected: "*completely bold*",
			},
			{
				name:     "Bold with special characters",
				input:    "**bold with !@#$%^&*() characters**",
				expected: "*bold with !@#$%^&*() characters*",
			},
			{
				name:     "Multiple lines with bold",
				input:    "First line with **bold**\nSecond line with **more bold**",
				expected: "First line with *bold*\nSecond line with *more bold*",
			},
			{
				name:     "Heading level 1",
				input:    "# Heading 1",
				expected: "*Heading 1*",
			},
			{
				name:     "Heading level 2",
				input:    "## Heading 2",
				expected: "*Heading 2*",
			},
			{
				name:     "Heading level 3",
				input:    "### Heading 3",
				expected: "*Heading 3*",
			},
			{
				name:     "Multiple headings",
				input:    "# First Heading\nSome text\n## Second Heading",
				expected: "*First Heading*\nSome text\n*Second Heading*",
			},
			{
				name:     "Heading without space after #",
				input:    "#NoSpace",
				expected: "*NoSpace*",
			},
			{
				name:     "Heading with extra spaces",
				input:    "##   Lots of spaces",
				expected: "*Lots of spaces*",
			},
			{
				name:     "Mixed bold and headings",
				input:    "# Main Title\nThis has **bold text** in it\n## Subtitle",
				expected: "*Main Title*\nThis has *bold text* in it\n*Subtitle*",
			},
			{
				name:     "Hashtag in middle of line (not heading)",
				input:    "This is not # a heading",
				expected: "This is not # a heading",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := ConvertMarkdownToSlack(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestAssertInvariant(t *testing.T) {
	t.Run("TrueCondition", func(t *testing.T) {
		// Should not panic
		assert.NotPanics(t, func() {
			AssertInvariant(true, "This should not panic")
		})
	})

	t.Run("FalseCondition", func(t *testing.T) {
		// Should panic with the correct message
		assert.PanicsWithValue(t, "invariant violated - This should panic", func() {
			AssertInvariant(false, "This should panic")
		})
	})

	t.Run("ComplexCondition", func(t *testing.T) {
		x := 5
		y := 10

		// Should not panic
		assert.NotPanics(t, func() {
			AssertInvariant(x < y, "x should be less than y")
		})

		// Should panic
		assert.PanicsWithValue(t, "invariant violated - x should be greater than y", func() {
			AssertInvariant(x > y, "x should be greater than y")
		})
	})
}

// MockSlackClient is a mock implementation of the Slack client for testing
type MockSlackClient struct {
	mock.Mock
}

func (m *MockSlackClient) GetUserInfoContext(ctx context.Context, user string) (*slack.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*slack.User), args.Error(1)
}

func TestResolveMentionsInSlackMessage(t *testing.T) {
	t.Run("NoMentions", func(t *testing.T) {
		mockClient := &MockSlackClient{}

		message := "This is a regular message with no mentions"
		result := ResolveMentionsInSlackMessage(context.Background(), message, mockClient)

		assert.Equal(t, message, result)
		mockClient.AssertNotCalled(t, "GetUserInfoContext")
	})

	t.Run("SingleMentionResolved", func(t *testing.T) {
		mockClient := &MockSlackClient{}

		// Setup mock user response
		mockUser := &slack.User{
			ID: "U123456",
			Profile: slack.UserProfile{
				DisplayName: "John Doe",
				RealName:    "John Smith Doe",
			},
		}
		mockClient.On("GetUserInfoContext", mock.Anything, "U123456").Return(mockUser, nil)

		message := "Hey <@U123456>, can you help with this?"
		result := ResolveMentionsInSlackMessage(context.Background(), message, mockClient)

		expected := "Hey @John Doe, can you help with this?"
		assert.Equal(t, expected, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("MultipleMentionsResolved", func(t *testing.T) {
		mockClient := &MockSlackClient{}

		// Setup mock user responses
		mockUser1 := &slack.User{
			ID: "U123456",
			Profile: slack.UserProfile{
				DisplayName: "John Doe",
			},
		}
		mockUser2 := &slack.User{
			ID: "U789012",
			Profile: slack.UserProfile{
				RealName: "Jane Smith",
			},
		}
		mockClient.On("GetUserInfoContext", mock.Anything, "U123456").Return(mockUser1, nil)
		mockClient.On("GetUserInfoContext", mock.Anything, "U789012").Return(mockUser2, nil)

		message := "Hey <@U123456> and <@U789012>, can you help?"
		result := ResolveMentionsInSlackMessage(context.Background(), message, mockClient)

		expected := "Hey @John Doe and @Jane Smith, can you help?"
		assert.Equal(t, expected, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("DuplicateMentions", func(t *testing.T) {
		mockClient := &MockSlackClient{}

		// Setup mock user response - should only be called once due to caching
		mockUser := &slack.User{
			ID: "U123456",
			Profile: slack.UserProfile{
				DisplayName: "John Doe",
			},
		}
		mockClient.On("GetUserInfoContext", mock.Anything, "U123456").Return(mockUser, nil).Once()

		message := "Hey <@U123456>, please tell <@U123456> about this"
		result := ResolveMentionsInSlackMessage(context.Background(), message, mockClient)

		expected := "Hey @John Doe, please tell @John Doe about this"
		assert.Equal(t, expected, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("APIError", func(t *testing.T) {
		mockClient := &MockSlackClient{}

		// Setup mock to return error
		mockClient.On("GetUserInfoContext", mock.Anything, "U123456").Return(nil, errors.New("user not found"))

		message := "Hey <@U123456>, can you help?"
		result := ResolveMentionsInSlackMessage(context.Background(), message, mockClient)

		// Should fall back to original mention format
		expected := "Hey <@U123456>, can you help?"
		assert.Equal(t, expected, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("BotMention", func(t *testing.T) {
		mockClient := &MockSlackClient{}

		// Setup mock bot user response
		mockUser := &slack.User{
			ID: "W123456",
			Profile: slack.UserProfile{
				DisplayName: "Bot Name",
			},
		}
		mockClient.On("GetUserInfoContext", mock.Anything, "W123456").Return(mockUser, nil)

		message := "Hey <@W123456>, can you help?"
		result := ResolveMentionsInSlackMessage(context.Background(), message, mockClient)

		expected := "Hey @Bot Name, can you help?"
		assert.Equal(t, expected, result)
		mockClient.AssertExpectations(t)
	})
}

func TestGetUserDisplayName(t *testing.T) {
	t.Run("DisplayNameAvailable", func(t *testing.T) {
		user := &slack.User{
			ID:   "U123",
			Name: "john.doe",
			Profile: slack.UserProfile{
				DisplayName: "John Doe",
				RealName:    "John Smith Doe",
			},
		}

		result := getUserDisplayName(user)
		assert.Equal(t, "John Doe", result)
	})

	t.Run("OnlyRealNameAvailable", func(t *testing.T) {
		user := &slack.User{
			ID:   "U123",
			Name: "john.doe",
			Profile: slack.UserProfile{
				RealName: "John Smith Doe",
			},
		}

		result := getUserDisplayName(user)
		assert.Equal(t, "John Smith Doe", result)
	})

	t.Run("OnlyUsernameAvailable", func(t *testing.T) {
		user := &slack.User{
			ID:   "U123",
			Name: "john.doe",
		}

		result := getUserDisplayName(user)
		assert.Equal(t, "john.doe", result)
	})

	t.Run("OnlyUserIDAvailable", func(t *testing.T) {
		user := &slack.User{
			ID: "U123",
		}

		result := getUserDisplayName(user)
		assert.Equal(t, "U123", result)
	})
}
