package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"
)

func NewConnection(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
