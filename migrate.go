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

// setupDatabase creates tables and seeds initial data
func setupDatabase() error {
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
	var db *sql.DB
	maxRetries := 60
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		db = stdlib.OpenDB(*config)
		if err := db.Ping(); err != nil {
			db.Close()
			if i < maxRetries-1 {
				log.Printf("Database not ready, retrying in %v... (attempt %d/%d)", retryDelay, i+1, maxRetries)
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
		}
		log.Println("Database connection established")
		break
	}
	defer db.Close()

	log.Println("Creating database schema...")
	if err := ensureSchema(db); err != nil {
		return err
	}

	log.Println("Schema created successfully")

	// Seed categories
	log.Println("Seeding categories...")
	if err := seedDefaultCategories(db); err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	log.Printf("Categories seeded successfully")

	return nil
}

// verifyDatabaseConnection tests the database connection
func verifyDatabaseConnection() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@postgres:5432/finance?sslmode=disable"
	}

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	db := stdlib.OpenDB(*config)
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection verified")
	return nil
}
