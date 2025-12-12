package db

import (
	"fmt"
	"os"
	"strings"
)

// Database is a wrapper that implements the DB interface
// It delegates to either SQLite or PostgreSQL implementation
type Database struct {
	DB
}

// NewDatabase creates a new database instance based on the provided configuration
// If databaseURL is provided (postgres://...), it uses PostgreSQL or TimescaleDB
// Otherwise, it falls back to SQLite with dbPath
func NewDatabase(dbPath string) (*Database, error) {
	// Check for DATABASE_URL environment variable first
	databaseURL := os.Getenv("DATABASE_URL")
	
	var impl DB
	var err error
	
	if databaseURL != "" && strings.HasPrefix(databaseURL, "postgres") {
		// Check if TimescaleDB is requested
		if os.Getenv("USE_TIMESCALE") == "true" || strings.Contains(databaseURL, "timescale") {
			impl, err = NewTimescaleDB(databaseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize timescale: %w", err)
			}
		} else {
			// Use PostgreSQL
			impl, err = NewPostgresDB(databaseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize postgres: %w", err)
			}
		}
	} else {
		// Use SQLite
		impl, err = NewSQLiteDB(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sqlite: %w", err)
		}
	}
	
	return &Database{DB: impl}, nil
}

// NewDatabaseWithURL creates a new database instance with explicit URL
// This is useful when you want to pass the database URL directly
func NewDatabaseWithURL(databaseURL string, sqlitePath string) (*Database, error) {
	return NewDatabaseWithURLOptions(databaseURL, sqlitePath, false)
}

// NewDatabaseWithURLOptions creates a new database instance with explicit URL and TimescaleDB option
func NewDatabaseWithURLOptions(databaseURL string, sqlitePath string, useTimescale bool) (*Database, error) {
	var impl DB
	var err error
	
	if databaseURL != "" && strings.HasPrefix(databaseURL, "postgres") {
		// Check if TimescaleDB is requested (from parameter, env var, or URL contains "timescale")
		shouldUseTimescale := useTimescale || os.Getenv("USE_TIMESCALE") == "true" || strings.Contains(databaseURL, "timescale")
		if shouldUseTimescale {
			impl, err = NewTimescaleDB(databaseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize timescale: %w", err)
			}
		} else {
			// Use PostgreSQL
			impl, err = NewPostgresDB(databaseURL)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize postgres: %w", err)
			}
		}
	} else {
		// Use SQLite
		impl, err = NewSQLiteDB(sqlitePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sqlite: %w", err)
		}
	}
	
	return &Database{DB: impl}, nil
}
