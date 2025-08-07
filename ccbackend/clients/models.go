package clients

// SlackAuthTestResponse represents the response from Slack's auth.test API
type SlackAuthTestResponse struct {
	UserID string
	TeamID string
}

// SlackPermalinkParameters represents parameters for generating a Slack permalink
type SlackPermalinkParameters struct {
	Channel string
	TS      string
}

// SlackUser represents a Slack user
type SlackUser struct {
	ID      string
	Name    string
	Profile SlackUserProfile
}

// SlackUserProfile represents a Slack user's profile information
type SlackUserProfile struct {
	DisplayName string
	RealName    string
}

// SlackPostMessageResponse represents the response from posting a message to Slack
type SlackPostMessageResponse struct {
	Channel   string
	Timestamp string
}

// SlackItemRef represents a reference to a Slack message item
type SlackItemRef struct {
	Channel   string
	Timestamp string
}

// SlackGetReactionsParameters represents parameters for getting reactions
type SlackGetReactionsParameters struct {
	// Add any specific parameters if needed in the future
}

// SlackItemReaction represents a reaction on a Slack message
type SlackItemReaction struct {
	Name  string
	Users []string
}

// SlackMessageOption represents an option for sending Slack messages
type SlackMessageOption interface {
	Apply(*SlackMessageConfig)
}

// SlackMessageConfig holds configuration for Slack messages
type SlackMessageConfig struct {
	Text        string
	AsUser      bool
	ThreadTS    string
	UnfurlLinks bool
	UnfurlMedia bool
}

// SlackMsgOptionText creates a message option with text
type SlackMsgOptionText struct {
	Text   string
	Escape bool
}

func (opt SlackMsgOptionText) Apply(config *SlackMessageConfig) {
	config.Text = opt.Text
}

// SlackMsgOptionTS creates a message option with thread timestamp
type SlackMsgOptionTS struct {
	Timestamp string
}

func (opt SlackMsgOptionTS) Apply(config *SlackMessageConfig) {
	config.ThreadTS = opt.Timestamp
}

// Helper functions to create message options (similar to slack-go/slack API)
func SlackMsgOptionTextHelper(text string, escape bool) SlackMessageOption {
	return SlackMsgOptionText{Text: text, Escape: escape}
}

func SlackMsgOptionTSHelper(timestamp string) SlackMessageOption {
	return SlackMsgOptionTS{Timestamp: timestamp}
}
