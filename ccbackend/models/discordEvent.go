package models

type DiscordMessageEvent struct {
	MessageID string
	ChannelID string
	ThreadID  string
	User      string
	Text      string
	Guild     string
}