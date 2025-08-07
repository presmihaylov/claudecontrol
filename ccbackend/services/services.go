package services

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"ccbackend/models"
)

// UsersService defines the interface for user-related operations
type UsersService interface {
	GetOrCreateUser(ctx context.Context, authProvider, authProviderID string) (*models.User, error)
}

// SlackIntegrationsService defines the interface for Slack integration operations
type SlackIntegrationsService interface {
	CreateSlackIntegration(ctx context.Context, slackAuthCode, redirectURL string, userID string) (*models.SlackIntegration, error)
	GetSlackIntegrationsByUserID(ctx context.Context, userID string) ([]*models.SlackIntegration, error)
	GetAllSlackIntegrations(ctx context.Context) ([]*models.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, integrationID string) error
	GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error)
	GetSlackIntegrationBySecretKey(ctx context.Context, secretKey string) (*models.SlackIntegration, error)
	GetSlackIntegrationByTeamID(ctx context.Context, teamID string) (*models.SlackIntegration, error)
	GetSlackIntegrationByID(ctx context.Context, id string) (*models.SlackIntegration, error)
}

// TransactionManager handles database transactions via context
type TransactionManager interface {
	// Execute function within a transaction (recommended approach)
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// Manual transaction control (for complex scenarios)
	BeginTransaction(ctx context.Context) (context.Context, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error
}

// Transactional interface that both *sqlx.DB and *sqlx.Tx implement
type Transactional interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
}
