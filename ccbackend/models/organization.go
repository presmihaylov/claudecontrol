package models

import (
	"time"
)

type Organization struct {
	ID        string    `json:"id"         db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
