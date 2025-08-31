package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"ccbackend/core"
	dbtx "ccbackend/db/tx"
	"ccbackend/models"
)

type PostgresSettingsRepository struct {
	db     *sqlx.DB
	schema string
}

// Column names for settings table
var settingsColumns = []string{
	"id",
	"organization_id",
	"scope_type",
	"scope_id",
	"key",
	"value_boolean",
	"value_string",
	"value_stringarr",
	"created_at",
	"updated_at",
}

func NewPostgresSettingsRepository(db *sqlx.DB, schema string) *PostgresSettingsRepository {
	return &PostgresSettingsRepository{db: db, schema: schema}
}

func (r *PostgresSettingsRepository) UpsertBooleanSetting(
	ctx context.Context,
	organizationID, scopeType, scopeID, key string,
	value bool,
) (*models.Setting, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	id := core.NewID("set")
	returningStr := strings.Join(settingsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.settings (
			id, organization_id, scope_type, scope_id, key, value_boolean
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (organization_id, scope_type, scope_id, key)
		DO UPDATE SET
			value_boolean = EXCLUDED.value_boolean,
			value_string = NULL,
			value_stringarr = NULL,
			updated_at = NOW()
		RETURNING %s
	`, r.schema, returningStr)

	var setting models.Setting
	err := db.QueryRowxContext(
		ctx,
		query,
		id, organizationID, scopeType, scopeID, key, value,
	).StructScan(&setting)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert boolean setting: %w", err)
	}

	return &setting, nil
}

func (r *PostgresSettingsRepository) UpsertStringSetting(
	ctx context.Context,
	organizationID, scopeType, scopeID, key string,
	value string,
) (*models.Setting, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	id := core.NewID("set")
	returningStr := strings.Join(settingsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.settings (
			id, organization_id, scope_type, scope_id, key, value_string
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (organization_id, scope_type, scope_id, key)
		DO UPDATE SET
			value_boolean = NULL,
			value_string = EXCLUDED.value_string,
			value_stringarr = NULL,
			updated_at = NOW()
		RETURNING %s
	`, r.schema, returningStr)

	var setting models.Setting
	err := db.QueryRowxContext(
		ctx,
		query,
		id, organizationID, scopeType, scopeID, key, value,
	).StructScan(&setting)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert string setting: %w", err)
	}

	return &setting, nil
}

func (r *PostgresSettingsRepository) UpsertStringArraySetting(
	ctx context.Context,
	organizationID, scopeType, scopeID, key string,
	value []string,
) (*models.Setting, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	id := core.NewID("set")
	returningStr := strings.Join(settingsColumns, ", ")

	query := fmt.Sprintf(`
		INSERT INTO %s.settings (
			id, organization_id, scope_type, scope_id, key, value_stringarr
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (organization_id, scope_type, scope_id, key)
		DO UPDATE SET
			value_boolean = NULL,
			value_string = NULL,
			value_stringarr = EXCLUDED.value_stringarr,
			updated_at = NOW()
		RETURNING %s
	`, r.schema, returningStr)

	var setting models.Setting
	err := db.QueryRowxContext(
		ctx,
		query,
		id, organizationID, scopeType, scopeID, key, pq.Array(value),
	).StructScan(&setting)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert string array setting: %w", err)
	}

	return &setting, nil
}

func (r *PostgresSettingsRepository) GetSetting(
	ctx context.Context,
	organizationID, scopeType, scopeID, key string,
) (*models.Setting, error) {
	db := dbtx.GetTransactional(ctx, r.db)

	returningStr := strings.Join(settingsColumns, ", ")

	query := fmt.Sprintf(`
		SELECT %s
		FROM %s.settings
		WHERE organization_id = $1
		  AND scope_type = $2
		  AND scope_id = $3
		  AND key = $4
	`, returningStr, r.schema)

	var setting models.Setting
	err := db.QueryRowxContext(
		ctx,
		query,
		organizationID, scopeType, scopeID, key,
	).StructScan(&setting)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("setting not found")
		}
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	return &setting, nil
}
