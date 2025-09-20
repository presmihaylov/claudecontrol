package models

import (
	"time"
)

// ConnectedChannel represents a tracked channel (Slack or Discord) with polymorphic support
type ConnectedChannel struct {
	ID             string    `json:"id"              db:"id"`
	OrgID          OrgID     `json:"organization_id" db:"organization_id"`
	ChannelID      string    `json:"channel_id"      db:"channel_id"`
	ChannelType    string    `json:"channel_type"    db:"channel_type"`    // "slack" or "discord"
	DefaultRepoURL *string   `json:"default_repo_url" db:"default_repo_url"` // Repository URL from first available agent
	CreatedAt      time.Time `json:"created_at"      db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"      db:"updated_at"`
}

// ChannelType constants
const (
	ChannelTypeSlack   = "slack"
	ChannelTypeDiscord = "discord"
)