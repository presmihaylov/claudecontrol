package models

import (
	"time"
)

type User struct {
	ID             string    `db:"id"               json:"id"`
	AuthProvider   string    `db:"auth_provider"    json:"auth_provider"`
	AuthProviderID string    `db:"auth_provider_id" json:"auth_provider_id"`
	CreatedAt      time.Time `db:"created_at"       json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"       json:"updated_at"`
}
