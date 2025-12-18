package db

import (
	"fmt"
	"os"
)

// Database is a wrapper that implements the DB interface
// It delegates to TimescaleDB implementation
type Database struct {
	DB
}

// NewDatabase creates a new database instance using TimescaleDB
// Requires DATABASE_URL environment variable to be set
func NewDatabase() (*Database, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	impl, err := NewTimescaleDB(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize timescale: %w", err)
	}

	return &Database{DB: impl}, nil
}

// NewDatabaseWithURL creates a new database instance with explicit URL
func NewDatabaseWithURL(databaseURL string) (*Database, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	impl, err := NewTimescaleDB(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize timescale: %w", err)
	}

	return &Database{DB: impl}, nil
}
