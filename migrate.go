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

	// Initialize schema
	schema := `
		CREATE TABLE IF NOT EXISTS categories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			type VARCHAR(20) NOT NULL,
			color VARCHAR(7) DEFAULT '#667eea',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS transactions (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL,
			description VARCHAR(255) NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			category_id INTEGER REFERENCES categories(id),
			type VARCHAR(20) NOT NULL,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS budgets (
			id SERIAL PRIMARY KEY,
			category_id INTEGER REFERENCES categories(id),
			amount DECIMAL(10,2) NOT NULL,
			period VARCHAR(20) DEFAULT 'monthly',
			start_date DATE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Remove duplicates before creating unique index
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.tables 
				WHERE table_schema = 'public' AND table_name = 'categories'
			) THEN
				WITH d AS (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY name, type ORDER BY id) rn
					FROM categories
				)
				DELETE FROM categories WHERE id IN (SELECT id FROM d WHERE rn > 1);
			END IF;
		END $$;

		-- Ensure uniqueness on (name, type)
		CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name_type ON categories(name, type);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Schema created successfully")

	// Seed categories
	log.Println("Seeding categories...")
	seedCategories := `
		INSERT INTO categories (name, type, color) VALUES
			('Groceries', 'expense', '#e74c3c'),
			('Rent', 'expense', '#e67e22'),
			('Utilities', 'expense', '#f39c12'),
			('Transportation', 'expense', '#3498db'),
			('Entertainment', 'expense', '#9b59b6'),
			('Salary', 'income', '#27ae60'),
			('Freelance', 'income', '#16a085')
		ON CONFLICT (name, type) DO NOTHING;
	`

	result, err := db.Exec(seedCategories)
	if err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Categories seeded successfully (%d rows affected)", rowsAffected)

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
