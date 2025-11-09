package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	db = stdlib.OpenDB(*config)

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

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
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Seed categories
	seedCategories := `
		INSERT INTO categories (name, type, color) VALUES
			('Groceries', 'expense', '#e74c3c'),
			('Rent', 'expense', '#e67e22'),
			('Utilities', 'expense', '#f39c12'),
			('Transportation', 'expense', '#3498db'),
			('Entertainment', 'expense', '#9b59b6'),
			('Salary', 'income', '#27ae60'),
			('Freelance', 'income', '#16a085')
		ON CONFLICT DO NOTHING;
	`

	if _, err := db.Exec(seedCategories); err != nil {
		log.Printf("Warning: failed to seed categories: %v", err)
	}

	return nil
}
