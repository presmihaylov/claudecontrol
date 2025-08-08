package appctx

import (
	"context"

	"ccbackend/models"
)

// Context key for storing user and organization entities
type contextKey string

const (
	UserContextKey         contextKey = "user"
	OrganizationContextKey contextKey = "organization"
)

// SetUser adds the user entity to the request context
func SetUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// GetUser extracts the user entity from the request context
func GetUser(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}

// SetOrganization adds the organization entity to the request context
func SetOrganization(ctx context.Context, org *models.Organization) context.Context {
	return context.WithValue(ctx, OrganizationContextKey, org)
}

// GetOrganization extracts the organization entity from the request context
func GetOrganization(ctx context.Context) (*models.Organization, bool) {
	org, ok := ctx.Value(OrganizationContextKey).(*models.Organization)
	return org, ok
}
