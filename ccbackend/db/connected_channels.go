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
	"channel_id",
	"channel_type",
	"default_repo_url",
	"created_at",
	"updated_at",
}

func NewPostgresConnectedChannelsRepository(db *sqlx.DB, schema string) *PostgresConnectedChannelsRepository {
	return &PostgresConnectedChannelsRepository{db: db, schema: schema}
}

func (r *PostgresConnectedChannelsRepository) UpsertConnectedChannel(ctx context.Context, channel *models.ConnectedChannel) error {
	insertColumns := []string{
		"id",
		"organization_id",
		"channel_id",
		"channel_type",
		"default_repo_url",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(connectedChannelsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.connected_channels (%s)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (organization_id, channel_id, channel_type)
		DO UPDATE SET
			updated_at = NOW()
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query,
		channel.ID,
		channel.OrgID,
		channel.ChannelID,
		channel.ChannelType,
		channel.DefaultRepoURL).
		StructScan(channel)
	if err != nil {
		return fmt.Errorf("failed to upsert connected channel: %w", err)
	}

	return nil
}

func (r *PostgresConnectedChannelsRepository) GetConnectedChannelByChannelID(
	ctx context.Context,
	orgID models.OrgID,
	channelID string,
	channelType string,
) (mo.Option[*models.ConnectedChannel], error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1 AND channel_id = $2 AND channel_type = $3`,
		columnsStr, r.schema)

	channel := &models.ConnectedChannel{}
	err := r.db.GetContext(ctx, channel, query, orgID, channelID, channelType)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ConnectedChannel](), nil
		}
		return mo.None[*models.ConnectedChannel](), fmt.Errorf("failed to get connected channel: %w", err)
	}

	return mo.Some(channel), nil
}

func (r *PostgresConnectedChannelsRepository) GetConnectedChannelByID(
	ctx context.Context,
	id string,
	orgID models.OrgID,
) (mo.Option[*models.ConnectedChannel], error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE id = $1 AND organization_id = $2`, columnsStr, r.schema)

	channel := &models.ConnectedChannel{}
	err := r.db.GetContext(ctx, channel, query, id, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.ConnectedChannel](), nil
		}
		return mo.None[*models.ConnectedChannel](), fmt.Errorf("failed to get connected channel: %w", err)
	}

	return mo.Some(channel), nil
}

func (r *PostgresConnectedChannelsRepository) GetConnectedChannelsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) ([]*models.ConnectedChannel, error) {
	columnsStr := strings.Join(connectedChannelsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.connected_channels
		WHERE organization_id = $1
		ORDER BY created_at ASC`, columnsStr, r.schema)

	var channels []*models.ConnectedChannel
	err := r.db.SelectContext(ctx, &channels, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected channels by organization: %w", err)
	}

	// Initialize empty slice if nil to avoid JSON null serialization
	if channels == nil {
		channels = []*models.ConnectedChannel{}
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