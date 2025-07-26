package appctx

import (
	"context"

	"ccbackend/models"
)

// Context key for storing user entity
type contextKey string

const UserContextKey contextKey = "user"

// SetUser adds the user entity to the request context
func SetUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// GetUser extracts the user entity from the request context
func GetUser(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}