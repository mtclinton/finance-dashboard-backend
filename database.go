package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB

// initDB initializes the PostgreSQL database connection and schema
func initDB() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@postgres:5432/finance?sslmode=disable"
	}

	// Normalize postgresql:// to postgres:// and ensure sslmode is set
	if databaseURL != "" {
		// Replace postgresql:// with postgres:// for compatibility
		if len(databaseURL) > 11 && databaseURL[:11] == "postgresql:" {
			databaseURL = "postgres" + databaseURL[10:]
		}
		// Add sslmode=disable if not present
		if !strings.Contains(databaseURL, "sslmode=") {
			separator := "?"
			if strings.Contains(databaseURL, "?") {
				separator = "&"
			}
			databaseURL = databaseURL + separator + "sslmode=disable"
		}
	}

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Wait for database to be ready with retries
	maxRetries := 60
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		db = stdlib.OpenDB(*config)
		if err := db.Ping(); err != nil {
			db.Close()
			if i < maxRetries-1 {
				// Log the actual error every 10 attempts
				if i%10 == 0 || i < 5 {
					log.Printf("Database not ready, retrying in %v... (attempt %d/%d) Error: %v", retryDelay, i+1, maxRetries, err)
				} else {
					log.Printf("Database not ready, retrying in %v... (attempt %d/%d)", retryDelay, i+1, maxRetries)
				}
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
		}
		log.Println("Database connection established")
		break
	}

	// Initialize schema
	if err := ensureSchema(db); err != nil {
		return err
	}

	// Seed categories
	if err := seedDefaultCategories(db); err != nil {
		log.Printf("Warning: failed to seed categories: %v", err)
	}

	return nil
}
