package models

import (
	"time"
)

type Organisation struct {
	ID        string    `db:"id"         json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
