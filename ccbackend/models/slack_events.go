package models

type SlackMessageEvent struct {
	Channel  string
	User     string
	Text     string
	TS       string
	ThreadTS string
	Team     string
}
