package models

type DiscordMessageEvent struct {
	GuildID   string
	ChannelID string
	MessageID string
	UserID    string
	Content   string
	// ThreadID for thread messages (nil for top-level messages)
	ThreadID  *string
	// Mentions contains the user IDs of all users mentioned in this message
	Mentions  []string
}

type DiscordReactionEvent struct {
	GuildID   string
	ChannelID string
	MessageID string
	UserID    string
	EmojiName string
	// ThreadID for thread reactions (nil for top-level channel reactions)
	ThreadID *string
}