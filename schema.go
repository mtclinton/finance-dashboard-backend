package main

import (
	"database/sql"
	"fmt"
)

const schemaSQL = `
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

	-- Remove duplicates before enforcing uniqueness
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

const seedSQL = `
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

func ensureSchema(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func seedDefaultCategories(db *sql.DB) error {
	if _, err := db.Exec(seedSQL); err != nil {
		return fmt.Errorf("failed to seed categories: %w", err)
	}
	return nil
}


