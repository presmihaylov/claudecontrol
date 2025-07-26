package api

import (
	"time"

	"github.com/google/uuid"
)

// UserModel represents the user data returned by the API
type UserModel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}