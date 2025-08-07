package slack

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"ccbackend/clients"
)

// testSlackClient implements clients.SlackClient for testing
type testSlackClient struct {
	getUserInfoFunc func(ctx context.Context, userID string) (*clients.SlackUser, error)
}

func (t *testSlackClient) GetOAuthV2Response(
	httpClient *http.Client,
	clientID, clientSecret, code, redirectURL string,
) (*clients.OAuthV2Response, error) {
	return nil, errors.New("not implemented")
}

func (t *testSlackClient) AuthTest() (*clients.SlackAuthTestResponse, error) {
	return nil, errors.New("not implemented")
}

func (t *testSlackClient) GetPermalink(params *clients.SlackPermalinkParameters) (string, error) {
	return "", errors.New("not implemented")
}

func (t *testSlackClient) GetUserInfoContext(ctx context.Context, userID string) (*clients.SlackUser, error) {
	if t.getUserInfoFunc != nil {
		return t.getUserInfoFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (t *testSlackClient) PostMessage(
	channelID string,
	params clients.SlackMessageParams,
) (*clients.SlackPostMessageResponse, error) {
	return nil, errors.New("not implemented")
}

func (t *testSlackClient) GetReactions(
	item clients.SlackItemRef,
	params clients.SlackGetReactionsParameters,
) ([]clients.SlackItemReaction, error) {
	return nil, errors.New("not implemented")
}

func (t *testSlackClient) AddReaction(name string, item clients.SlackItemRef) error {
	return errors.New("not implemented")
}

func (t *testSlackClient) RemoveReaction(name string, item clients.SlackItemRef) error {
	return errors.New("not implemented")
}

func (t *testSlackClient) ResolveMentionsInMessage(ctx context.Context, message string) string {
	return message
}

func TestResolveMentionsInMessage(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		mockSetup       func(*testSlackClient)
		expectedMessage string
	}{
		{
			name:            "no mentions in message",
			message:         "Hello world! This is a regular message.",
			mockSetup:       func(mock *testSlackClient) {},
			expectedMessage: "Hello world! This is a regular message.",
		},
		{
			name:    "single user mention resolved successfully",
			message: "Hello <@U123456>! How are you?",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					if userID == "U123456" {
						return &clients.SlackUser{
							ID:   "U123456",
							Name: "john.doe",
							Profile: clients.SlackUserProfile{
								DisplayName: "John Doe",
								RealName:    "John Doe",
							},
						}, nil
					}
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @John Doe! How are you?",
		},
		{
			name:    "multiple user mentions resolved successfully",
			message: "Hello <@U123456> and <@U789012>! How are you both?",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					if userID == "U123456" {
						return &clients.SlackUser{
							ID:   "U123456",
							Name: "john.doe",
							Profile: clients.SlackUserProfile{
								DisplayName: "John Doe",
								RealName:    "John Doe",
							},
						}, nil
					}
					if userID == "U789012" {
						return &clients.SlackUser{
							ID:   "U789012",
							Name: "jane.smith",
							Profile: clients.SlackUserProfile{
								DisplayName: "Jane Smith",
								RealName:    "Jane Smith",
							},
						}, nil
					}
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @John Doe and @Jane Smith! How are you both?",
		},
		{
			name:    "duplicate user mentions resolved only once",
			message: "Hello <@U123456>! <@U123456>, are you there?",
			mockSetup: func(mock *testSlackClient) {
				callCount := 0
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					callCount++
					if userID == "U123456" {
						return &clients.SlackUser{
							ID:   "U123456",
							Name: "john.doe",
							Profile: clients.SlackUserProfile{
								DisplayName: "John Doe",
								RealName:    "John Doe",
							},
						}, nil
					}
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @John Doe! @John Doe, are you there?",
		},
		{
			name:    "user mention API call fails - keeps original format",
			message: "Hello <@U999999>! How are you?",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					return nil, errors.New("API error")
				}
			},
			expectedMessage: "Hello <@U999999>! How are you?",
		},
		{
			name:    "user with only RealName (no DisplayName)",
			message: "Hello <@U123456>!",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					if userID == "U123456" {
						return &clients.SlackUser{
							ID:   "U123456",
							Name: "john.doe",
							Profile: clients.SlackUserProfile{
								DisplayName: "", // Empty display name
								RealName:    "John Doe",
							},
						}, nil
					}
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @John Doe!",
		},
		{
			name:    "user with only Name (no DisplayName or RealName)",
			message: "Hello <@U123456>!",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					if userID == "U123456" {
						return &clients.SlackUser{
							ID:   "U123456",
							Name: "john.doe",
							Profile: clients.SlackUserProfile{
								DisplayName: "", // Empty display name
								RealName:    "", // Empty real name
							},
						}, nil
					}
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @john.doe!",
		},
		{
			name:    "user with only ID (no other names)",
			message: "Hello <@U123456>!",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					if userID == "U123456" {
						return &clients.SlackUser{
							ID:   "U123456",
							Name: "", // Empty name
							Profile: clients.SlackUserProfile{
								DisplayName: "", // Empty display name
								RealName:    "", // Empty real name
							},
						}, nil
					}
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @U123456!",
		},
		{
			name:    "workspace user mention (starts with W)",
			message: "Hello <@W123456>!",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					if userID == "W123456" {
						return &clients.SlackUser{
							ID:   "W123456",
							Name: "workspace.bot",
							Profile: clients.SlackUserProfile{
								DisplayName: "Workspace Bot",
								RealName:    "Workspace Bot",
							},
						}, nil
					}
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @Workspace Bot!",
		},
		{
			name:    "mixed success and failure mentions",
			message: "Hello <@U123456> and <@U999999>!",
			mockSetup: func(mock *testSlackClient) {
				mock.getUserInfoFunc = func(ctx context.Context, userID string) (*clients.SlackUser, error) {
					if userID == "U123456" {
						return &clients.SlackUser{
							ID:   "U123456",
							Name: "john.doe",
							Profile: clients.SlackUserProfile{
								DisplayName: "John Doe",
								RealName:    "John Doe",
							},
						}, nil
					}
					// U999999 will fail
					return nil, errors.New("user not found")
				}
			},
			expectedMessage: "Hello @John Doe and <@U999999>!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Slack client
			mockClient := &testSlackClient{}
			tt.mockSetup(mockClient)

			// Call the function under test
			ctx := context.Background()
			result := ResolveMentionsInMessage(ctx, mockClient, tt.message)

			// Assert the result
			assert.Equal(t, tt.expectedMessage, result)
		})
	}
}

func TestGetUserDisplayName(t *testing.T) {
	tests := []struct {
		name         string
		user         *clients.SlackUser
		expectedName string
	}{
		{
			name: "display name takes priority",
			user: &clients.SlackUser{
				ID:   "U123456",
				Name: "john.doe",
				Profile: clients.SlackUserProfile{
					DisplayName: "John Doe",
					RealName:    "John Real Doe",
				},
			},
			expectedName: "John Doe",
		},
		{
			name: "real name used when display name is empty",
			user: &clients.SlackUser{
				ID:   "U123456",
				Name: "john.doe",
				Profile: clients.SlackUserProfile{
					DisplayName: "",
					RealName:    "John Real Doe",
				},
			},
			expectedName: "John Real Doe",
		},
		{
			name: "name used when both display name and real name are empty",
			user: &clients.SlackUser{
				ID:   "U123456",
				Name: "john.doe",
				Profile: clients.SlackUserProfile{
					DisplayName: "",
					RealName:    "",
				},
			},
			expectedName: "john.doe",
		},
		{
			name: "ID used when all other names are empty",
			user: &clients.SlackUser{
				ID:   "U123456",
				Name: "",
				Profile: clients.SlackUserProfile{
					DisplayName: "",
					RealName:    "",
				},
			},
			expectedName: "U123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUserDisplayName(tt.user)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}
