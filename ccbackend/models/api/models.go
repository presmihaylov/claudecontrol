package api

import (
	"time"
)

// UserModel represents the user data returned by the API
type UserModel struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
