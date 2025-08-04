package models

type SlackMessageEvent struct {
	Channel  string
	User     string
	Text     string
	Ts       string
	ThreadTs string
}
