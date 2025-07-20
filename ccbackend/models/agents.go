package models

import (
	"time"

	"github.com/google/uuid"
)

type ActiveAgent struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	AssignedJobID    *uuid.UUID `json:"assigned_job_id" db:"assigned_job_id"`
	WSConnectionID   string     `json:"ws_connection_id" db:"ws_connection_id"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}