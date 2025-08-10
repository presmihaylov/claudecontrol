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

type PostgresDiscordIntegrationsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for discord_integrations table
var discordIntegrationsColumns = []string{
	"id",
	"discord_guild_id",
	"discord_guild_name",
	"organization_id",
	"created_at",
	"updated_at",
}

func NewPostgresDiscordIntegrationsRepository(db *sqlx.DB, schema string) *PostgresDiscordIntegrationsRepository {
	return &PostgresDiscordIntegrationsRepository{db: db, schema: schema}
}

func (r *PostgresDiscordIntegrationsRepository) CreateDiscordIntegration(
	ctx context.Context,
	integration *models.DiscordIntegration,
) error {
	insertColumns := []string{
		"id",
		"discord_guild_id",
		"discord_guild_name",
		"organization_id",
		"created_at",
		"updated_at",
	}
	columnsStr := strings.Join(insertColumns, ", ")
	returningStr := strings.Join(discordIntegrationsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.discord_integrations (%s) 
		VALUES ($1, $2, $3, $4, NOW(), NOW()) 
		RETURNING %s`, r.schema, columnsStr, returningStr)

	err := r.db.QueryRowxContext(ctx, query, integration.ID, integration.DiscordGuildID, integration.DiscordGuildName, integration.OrganizationID).
		StructScan(integration)
	if err != nil {
		return fmt.Errorf("failed to create discord integration: %w", err)
	}

	return nil
}

func (r *PostgresDiscordIntegrationsRepository) GetDiscordIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID models.OrganizationID,
) ([]*models.DiscordIntegration, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	columnsStr := strings.Join(discordIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.discord_integrations 
		WHERE organization_id = $1 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	var integrations []*models.DiscordIntegration
	err := r.db.SelectContext(ctx, &integrations, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord integrations by organization ID: %w", err)
	}

	return integrations, nil
}

func (r *PostgresDiscordIntegrationsRepository) GetAllDiscordIntegrations(
	ctx context.Context,
) ([]*models.DiscordIntegration, error) {
	columnsStr := strings.Join(discordIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.discord_integrations 
		ORDER BY created_at DESC`, columnsStr, r.schema)

	var integrations []*models.DiscordIntegration
	err := r.db.SelectContext(ctx, &integrations, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all discord integrations: %w", err)
	}

	return integrations, nil
}

func (r *PostgresDiscordIntegrationsRepository) DeleteDiscordIntegrationByID(
	ctx context.Context,
	integrationID string,
	organizationID models.OrganizationID,
) (bool, error) {
	query := fmt.Sprintf(`DELETE FROM %s.discord_integrations WHERE id = $1 AND organization_id = $2`, r.schema)

	result, err := r.db.ExecContext(ctx, query, integrationID, organizationID)
	if err != nil {
		return false, fmt.Errorf("failed to delete discord integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *PostgresDiscordIntegrationsRepository) GetDiscordIntegrationByGuildID(
	ctx context.Context,
	guildID string,
) (mo.Option[*models.DiscordIntegration], error) {
	if guildID == "" {
		return mo.None[*models.DiscordIntegration](), fmt.Errorf("guild ID cannot be empty")
	}

	columnsStr := strings.Join(discordIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.discord_integrations 
		WHERE discord_guild_id = $1`, columnsStr, r.schema)

	var integration models.DiscordIntegration
	err := r.db.GetContext(ctx, &integration, query, guildID)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.DiscordIntegration](), nil
		}
		return mo.None[*models.DiscordIntegration](), fmt.Errorf(
			"failed to get discord integration by guild ID: %w",
			err,
		)
	}

	return mo.Some(&integration), nil
}

func (r *PostgresDiscordIntegrationsRepository) GetDiscordIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.DiscordIntegration], error) {
	columnsStr := strings.Join(discordIntegrationsColumns, ", ")
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s.discord_integrations 
		WHERE id = $1`, columnsStr, r.schema)

	var integration models.DiscordIntegration
	err := r.db.GetContext(ctx, &integration, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return mo.None[*models.DiscordIntegration](), nil
		}
		return mo.None[*models.DiscordIntegration](), fmt.Errorf("failed to get discord integration by ID: %w", err)
	}

	return mo.Some(&integration), nil
}
