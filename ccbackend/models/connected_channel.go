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

// DatabaseConnectedChannel represents the raw database record with all platform fields
type DatabaseConnectedChannel struct {
	ID                string    `json:"id"                 db:"id"`
	OrgID             OrgID     `json:"organization_id"    db:"organization_id"`
	SlackTeamID       *string   `json:"slack_team_id"      db:"slack_team_id"`
	SlackChannelID    *string   `json:"slack_channel_id"   db:"slack_channel_id"`
	DiscordGuildID    *string   `json:"discord_guild_id"   db:"discord_guild_id"`
	DiscordChannelID  *string   `json:"discord_channel_id" db:"discord_channel_id"`
	DefaultRepoURL    *string   `json:"default_repo_url"   db:"default_repo_url"`
	CreatedAt         time.Time `json:"created_at"         db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"         db:"updated_at"`
}

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

// Mapping functions

// ToSlackConnectedChannel converts a DatabaseConnectedChannel to SlackConnectedChannel
func (db *DatabaseConnectedChannel) ToSlackConnectedChannel() (*SlackConnectedChannel, error) {
	if db.SlackTeamID == nil || db.SlackChannelID == nil {
		return nil, fmt.Errorf("cannot convert to Slack channel: missing Slack fields")
	}
	if db.DiscordGuildID != nil || db.DiscordChannelID != nil {
		return nil, fmt.Errorf("cannot convert to Slack channel: Discord fields are populated")
	}

	return &SlackConnectedChannel{
		ID:             db.ID,
		OrgID:          db.OrgID,
		TeamID:         *db.SlackTeamID,
		ChannelID:      *db.SlackChannelID,
		DefaultRepoURL: db.DefaultRepoURL,
		CreatedAt:      db.CreatedAt,
		UpdatedAt:      db.UpdatedAt,
	}, nil
}

// ToDiscordConnectedChannel converts a DatabaseConnectedChannel to DiscordConnectedChannel
func (db *DatabaseConnectedChannel) ToDiscordConnectedChannel() (*DiscordConnectedChannel, error) {
	if db.DiscordGuildID == nil || db.DiscordChannelID == nil {
		return nil, fmt.Errorf("cannot convert to Discord channel: missing Discord fields")
	}
	if db.SlackTeamID != nil || db.SlackChannelID != nil {
		return nil, fmt.Errorf("cannot convert to Discord channel: Slack fields are populated")
	}

	return &DiscordConnectedChannel{
		ID:             db.ID,
		OrgID:          db.OrgID,
		GuildID:        *db.DiscordGuildID,
		ChannelID:      *db.DiscordChannelID,
		DefaultRepoURL: db.DefaultRepoURL,
		CreatedAt:      db.CreatedAt,
		UpdatedAt:      db.UpdatedAt,
	}, nil
}

// ToConnectedChannel converts a DatabaseConnectedChannel to the appropriate ConnectedChannel interface
func (db *DatabaseConnectedChannel) ToConnectedChannel() (ConnectedChannel, error) {
	if db.SlackTeamID != nil && db.SlackChannelID != nil {
		return db.ToSlackConnectedChannel()
	}
	if db.DiscordGuildID != nil && db.DiscordChannelID != nil {
		return db.ToDiscordConnectedChannel()
	}
	return nil, fmt.Errorf("invalid database record: no platform fields populated")
}

// GetChannelType returns the channel type based on populated fields
func (db *DatabaseConnectedChannel) GetChannelType() (ChannelType, error) {
	if db.SlackTeamID != nil && db.SlackChannelID != nil {
		return ChannelTypeSlack, nil
	}
	if db.DiscordGuildID != nil && db.DiscordChannelID != nil {
		return ChannelTypeDiscord, nil
	}
	return "", fmt.Errorf("invalid database record: no platform fields populated")
}

// Conversion functions from domain models back to database models

// FromSlackConnectedChannel creates a DatabaseConnectedChannel from SlackConnectedChannel
func FromSlackConnectedChannel(slack *SlackConnectedChannel) *DatabaseConnectedChannel {
	return &DatabaseConnectedChannel{
		ID:               slack.ID,
		OrgID:            slack.OrgID,
		SlackTeamID:      &slack.TeamID,
		SlackChannelID:   &slack.ChannelID,
		DiscordGuildID:   nil,
		DiscordChannelID: nil,
		DefaultRepoURL:   slack.DefaultRepoURL,
		CreatedAt:        slack.CreatedAt,
		UpdatedAt:        slack.UpdatedAt,
	}
}

// FromDiscordConnectedChannel creates a DatabaseConnectedChannel from DiscordConnectedChannel
func FromDiscordConnectedChannel(discord *DiscordConnectedChannel) *DatabaseConnectedChannel {
	return &DatabaseConnectedChannel{
		ID:               discord.ID,
		OrgID:            discord.OrgID,
		SlackTeamID:      nil,
		SlackChannelID:   nil,
		DiscordGuildID:   &discord.GuildID,
		DiscordChannelID: &discord.ChannelID,
		DefaultRepoURL:   discord.DefaultRepoURL,
		CreatedAt:        discord.CreatedAt,
		UpdatedAt:        discord.UpdatedAt,
	}
}