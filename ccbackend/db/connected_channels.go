package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/mo"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/models"
)

// DatabaseConnectedChannel represents the raw database record with all platform fields
type DatabaseConnectedChannel struct {
	ID                string    `json:"id"                 db:"id"`
	OrgID             models.OrgID `json:"organization_id"    db:"organization_id"`
	SlackTeamID       *string   `json:"slack_team_id"      db:"slack_team_id"`
	SlackChannelID    *string   `json:"slack_channel_id"   db:"slack_channel_id"`
	DiscordGuildID    *string   `json:"discord_guild_id"   db:"discord_guild_id"`
	DiscordChannelID  *string   `json:"discord_channel_id" db:"discord_channel_id"`
	DefaultRepoURL    *string   `json:"default_repo_url"   db:"default_repo_url"`
	CreatedAt         time.Time `json:"created_at"         db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"         db:"updated_at"`
}

// Mapping functions

// ToSlackConnectedChannel converts a DatabaseConnectedChannel to SlackConnectedChannel
func (db *DatabaseConnectedChannel) ToSlackConnectedChannel() (*models.SlackConnectedChannel, error) {
	if db.SlackTeamID == nil || db.SlackChannelID == nil {
		return nil, fmt.Errorf("cannot convert to Slack channel: missing Slack fields")
	}
	if db.DiscordGuildID != nil || db.DiscordChannelID != nil {
		return nil, fmt.Errorf("cannot convert to Slack channel: Discord fields are populated")
	}

	return &models.SlackConnectedChannel{
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
func (db *DatabaseConnectedChannel) ToDiscordConnectedChannel() (*models.DiscordConnectedChannel, error) {
	if db.DiscordGuildID == nil || db.DiscordChannelID == nil {
		return nil, fmt.Errorf("cannot convert to Discord channel: missing Discord fields")
	}
	if db.SlackTeamID != nil || db.SlackChannelID != nil {
		return nil, fmt.Errorf("cannot convert to Discord channel: Slack fields are populated")
	}

	return &models.DiscordConnectedChannel{
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
func (db *DatabaseConnectedChannel) ToConnectedChannel() (models.ConnectedChannel, error) {
	if db.SlackTeamID != nil && db.SlackChannelID != nil {
		return db.ToSlackConnectedChannel()
	}
	if db.DiscordGuildID != nil && db.DiscordChannelID != nil {
		return db.ToDiscordConnectedChannel()
	}
	return nil, fmt.Errorf("invalid database record: no platform fields populated")
}

// GetChannelType returns the channel type based on populated fields
func (db *DatabaseConnectedChannel) GetChannelType() (models.ChannelType, error) {
	if db.SlackTeamID != nil && db.SlackChannelID != nil {
		return models.ChannelTypeSlack, nil
	}
	if db.DiscordGuildID != nil && db.DiscordChannelID != nil {
		return models.ChannelTypeDiscord, nil
	}
	return "", fmt.Errorf("invalid database record: no platform fields populated")
}

// Conversion functions from domain models back to database models

// FromSlackConnectedChannel creates a DatabaseConnectedChannel from SlackConnectedChannel
func FromSlackConnectedChannel(slack *models.SlackConnectedChannel) *DatabaseConnectedChannel {
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
func FromDiscordConnectedChannel(discord *models.DiscordConnectedChannel) *DatabaseConnectedChannel {
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

type PostgresConnectedChannelsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for connected_channels table
var connectedChannelsColumns = []string{
	"id",
	"organization_id",
	"slack_team_id",
	"slack_channel_id",
	"discord_guild_id",
	"discord_channel_id",
	"default_repo_url",
	"created_at",
	"updated_at",
}

func NewPostgresConnectedChannelsRepository(db *sqlx.DB, schema string) *PostgresConnectedChannelsRepository {
	return &PostgresConnectedChannelsRepository{db: db, schema: schema}
}

func (r *PostgresConnectedChannelsRepository) UpsertSlackConnectedChannel(
	ctx context.Context,
	channel *DatabaseConnectedChannel,
) error {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	returningStr := strings.Join(connectedChannelsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.connected_channels (%s)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (organization_id, slack_team_id, slack_channel_id)
		DO UPDATE SET
			default_repo_url = EXCLUDED.default_repo_url,
			updated_at = NOW()
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query,
		channel.ID,
		channel.OrgID,
		channel.SlackTeamID,
		channel.SlackChannelID,
		channel.DiscordGuildID,
		channel.DiscordChannelID,
		channel.DefaultRepoURL).
		StructScan(channel)
	if err != nil {
		return fmt.Errorf("failed to upsert Slack connected channel: %w", err)
	}

	return nil
}

func (r *PostgresConnectedChannelsRepository) UpsertDiscordConnectedChannel(
	ctx context.Context,
	channel *DatabaseConnectedChannel,
) error {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	returningStr := strings.Join(connectedChannelsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.connected_channels (%s)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (organization_id, discord_guild_id, discord_channel_id)
		DO UPDATE SET
			default_repo_url = EXCLUDED.default_repo_url,
			updated_at = NOW()
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query,
		channel.ID,
		channel.OrgID,
		channel.SlackTeamID,
		channel.SlackChannelID,
		channel.DiscordGuildID,
		channel.DiscordChannelID,
		channel.DefaultRepoURL).
		StructScan(channel)
	if err != nil {
		return fmt.Errorf("failed to upsert Discord connected channel: %w", err)
	}

	return nil
}

func (r *PostgresConnectedChannelsRepository) GetSlackConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	teamID string,
	channelID string,
) (mo.Option[*DatabaseConnectedChannel], error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1 AND slack_team_id = $2 AND slack_channel_id = $3`,
		columnsStr, r.schema)

	channel := &DatabaseConnectedChannel{}
	err := r.db.GetContext(ctx, channel, query, orgID, teamID, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*DatabaseConnectedChannel](), nil
		}
		return mo.None[*DatabaseConnectedChannel](), fmt.Errorf("failed to get Slack connected channel: %w", err)
	}

	return mo.Some(channel), nil
}

func (r *PostgresConnectedChannelsRepository) GetDiscordConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
) (mo.Option[*DatabaseConnectedChannel], error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1 AND discord_guild_id = $2 AND discord_channel_id = $3`,
		columnsStr, r.schema)

	channel := &DatabaseConnectedChannel{}
	err := r.db.GetContext(ctx, channel, query, orgID, guildID, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*DatabaseConnectedChannel](), nil
		}
		return mo.None[*DatabaseConnectedChannel](), fmt.Errorf("failed to get Discord connected channel: %w", err)
	}

	return mo.Some(channel), nil
}







