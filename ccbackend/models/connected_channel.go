package models

import (
	"fmt"
	"time"
)

// ChannelType represents the type of channel
type ChannelType string

const (
	ChannelTypeSlack   ChannelType = "slack"
	ChannelTypeDiscord ChannelType = "discord"
)


// ConnectedChannel is the common interface for all connected channel types
type ConnectedChannel interface {
	GetID() string
	GetOrgID() OrgID
	GetChannelType() ChannelType
	GetDefaultRepoURL() *string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	// Platform-specific identifier for uniqueness
	GetPlatformIdentifier() string
}

// SlackConnectedChannel represents a connected Slack channel
type SlackConnectedChannel struct {
	ID             string    `json:"id"`
	OrgID          OrgID     `json:"organization_id"`
	TeamID         string    `json:"team_id"`
	ChannelID      string    `json:"channel_id"`
	DefaultRepoURL *string   `json:"default_repo_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GetID implements ConnectedChannel interface
func (s *SlackConnectedChannel) GetID() string {
	return s.ID
}

// GetOrgID implements ConnectedChannel interface
func (s *SlackConnectedChannel) GetOrgID() OrgID {
	return s.OrgID
}

// GetChannelType implements ConnectedChannel interface
func (s *SlackConnectedChannel) GetChannelType() ChannelType {
	return ChannelTypeSlack
}

// GetDefaultRepoURL implements ConnectedChannel interface
func (s *SlackConnectedChannel) GetDefaultRepoURL() *string {
	return s.DefaultRepoURL
}

// GetCreatedAt implements ConnectedChannel interface
func (s *SlackConnectedChannel) GetCreatedAt() time.Time {
	return s.CreatedAt
}

// GetUpdatedAt implements ConnectedChannel interface
func (s *SlackConnectedChannel) GetUpdatedAt() time.Time {
	return s.UpdatedAt
}

// GetPlatformIdentifier implements ConnectedChannel interface
func (s *SlackConnectedChannel) GetPlatformIdentifier() string {
	return fmt.Sprintf("slack:%s:%s", s.TeamID, s.ChannelID)
}

// DiscordConnectedChannel represents a connected Discord channel
type DiscordConnectedChannel struct {
	ID             string    `json:"id"`
	OrgID          OrgID     `json:"organization_id"`
	GuildID        string    `json:"guild_id"`
	ChannelID      string    `json:"channel_id"`
	DefaultRepoURL *string   `json:"default_repo_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GetID implements ConnectedChannel interface
func (d *DiscordConnectedChannel) GetID() string {
	return d.ID
}

// GetOrgID implements ConnectedChannel interface
func (d *DiscordConnectedChannel) GetOrgID() OrgID {
	return d.OrgID
}

// GetChannelType implements ConnectedChannel interface
func (d *DiscordConnectedChannel) GetChannelType() ChannelType {
	return ChannelTypeDiscord
}

// GetDefaultRepoURL implements ConnectedChannel interface
func (d *DiscordConnectedChannel) GetDefaultRepoURL() *string {
	return d.DefaultRepoURL
}

// GetCreatedAt implements ConnectedChannel interface
func (d *DiscordConnectedChannel) GetCreatedAt() time.Time {
	return d.CreatedAt
}

// GetUpdatedAt implements ConnectedChannel interface
func (d *DiscordConnectedChannel) GetUpdatedAt() time.Time {
	return d.UpdatedAt
}

// GetPlatformIdentifier implements ConnectedChannel interface
func (d *DiscordConnectedChannel) GetPlatformIdentifier() string {
	return fmt.Sprintf("discord:%s:%s", d.GuildID, d.ChannelID)
}


