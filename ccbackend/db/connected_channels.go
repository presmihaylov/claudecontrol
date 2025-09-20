package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/samber/mo"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"

	"ccbackend/models"
)

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
	channel *models.DatabaseConnectedChannel,
) error {
	insertColumns := []string{
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
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(connectedChannelsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.connected_channels (%s)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (organization_id, slack_team_id, slack_channel_id)
		DO UPDATE SET
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
	channel *models.DatabaseConnectedChannel,
) error {
	insertColumns := []string{
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
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(connectedChannelsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.connected_channels (%s)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (organization_id, discord_guild_id, discord_channel_id)
		DO UPDATE SET
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
) (mo.Option[*models.DatabaseConnectedChannel], error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1 AND slack_team_id = $2 AND slack_channel_id = $3`,
		columnsStr, r.schema)

	channel := &models.DatabaseConnectedChannel{}
	err := r.db.GetContext(ctx, channel, query, orgID, teamID, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.DatabaseConnectedChannel](), nil
		}
		return mo.None[*models.DatabaseConnectedChannel](), fmt.Errorf("failed to get Slack connected channel: %w", err)
	}

	return mo.Some(channel), nil
}

func (r *PostgresConnectedChannelsRepository) GetDiscordConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
) (mo.Option[*models.DatabaseConnectedChannel], error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1 AND discord_guild_id = $2 AND discord_channel_id = $3`,
		columnsStr, r.schema)

	channel := &models.DatabaseConnectedChannel{}
	err := r.db.GetContext(ctx, channel, query, orgID, guildID, channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.DatabaseConnectedChannel](), nil
		}
		return mo.None[*models.DatabaseConnectedChannel](), fmt.Errorf("failed to get Discord connected channel: %w", err)
	}

	return mo.Some(channel), nil
}

func (r *PostgresConnectedChannelsRepository) GetConnectedChannelByID(
	ctx context.Context,
	id string,
	orgID models.OrgID,
) (mo.Option[*models.DatabaseConnectedChannel], error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	channel := &models.DatabaseConnectedChannel{}
	err := r.db.GetContext(ctx, channel, query, id, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.DatabaseConnectedChannel](), nil
		}
		return mo.None[*models.DatabaseConnectedChannel](), fmt.Errorf("failed to get connected channel: %w", err)
	}

	return mo.Some(channel), nil
}

func (r *PostgresConnectedChannelsRepository) GetConnectedChannelsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) ([]*models.DatabaseConnectedChannel, error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1
		ORDER BY created_at ASC`, columnsStr, r.schema)

	var channels []*models.DatabaseConnectedChannel
	err := r.db.SelectContext(ctx, &channels, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected channels by organization: %w", err)
	}

	// Initialize empty slice if nil to avoid JSON null serialization
	if channels == nil {
		channels = []*models.DatabaseConnectedChannel{}
	}

	return channels, nil
}

func (r *PostgresConnectedChannelsRepository) GetSlackConnectedChannelsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) ([]*models.DatabaseConnectedChannel, error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1 AND slack_team_id IS NOT NULL
		ORDER BY created_at ASC`, columnsStr, r.schema)

	var channels []*models.DatabaseConnectedChannel
	err := r.db.SelectContext(ctx, &channels, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Slack connected channels by organization: %w", err)
	}

	// Initialize empty slice if nil to avoid JSON null serialization
	if channels == nil {
		channels = []*models.DatabaseConnectedChannel{}
	}

	return channels, nil
}

func (r *PostgresConnectedChannelsRepository) GetDiscordConnectedChannelsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) ([]*models.DatabaseConnectedChannel, error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1 AND discord_guild_id IS NOT NULL
		ORDER BY created_at ASC`, columnsStr, r.schema)

	var channels []*models.DatabaseConnectedChannel
	err := r.db.SelectContext(ctx, &channels, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord connected channels by organization: %w", err)
	}

	// Initialize empty slice if nil to avoid JSON null serialization
	if channels == nil {
		channels = []*models.DatabaseConnectedChannel{}
	}

	return channels, nil
}

func (r *PostgresConnectedChannelsRepository) DeleteConnectedChannel(
	ctx context.Context,
	id string,
	orgID models.OrgID,
) (bool, error) {
	query := fmt.Sprintf("DELETE FROM %s.connected_channels WHERE id = $1 AND organization_id = $2", r.schema)

	result, err := r.db.ExecContext(ctx, query, id, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to delete connected channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresConnectedChannelsRepository) UpdateConnectedChannelDefaultRepoURL(
	ctx context.Context,
	id string,
	orgID models.OrgID,
	defaultRepoURL *string,
) (bool, error) {
	query := fmt.Sprintf(`
		UPDATE %s.connected_channels
		SET default_repo_url = $1, updated_at = NOW()
		WHERE id = $2 AND organization_id = $3`, r.schema)

	result, err := r.db.ExecContext(ctx, query, defaultRepoURL, id, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to update connected channel default repo URL: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}