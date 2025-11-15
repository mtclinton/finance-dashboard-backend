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

// Seed a small set of demo transactions and budgets for presentations.
// Idempotent: will only run if there are zero transactions present.
func seedDemoData(db *sql.DB) error {
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&cnt); err != nil {
		return fmt.Errorf("checking transactions count: %w", err)
	}
	if cnt > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Insert a handful of income/expense demo transactions over the last ~30 days
	// Categories assumed to exist from seedDefaultCategories.
	const demoTx = `
	INSERT INTO transactions (date, description, amount, category_id, type, notes) VALUES
	(CURRENT_DATE - INTERVAL '28 days', 'Monthly Salary', 3200.00, (SELECT id FROM categories WHERE name='Salary' AND type='income' LIMIT 1), 'income', 'November payroll'),
	(CURRENT_DATE - INTERVAL '25 days', 'Freelance: Landing Page', 850.00, (SELECT id FROM categories WHERE name='Freelance' AND type='income' LIMIT 1), 'income', 'Side project'),
	(CURRENT_DATE - INTERVAL '24 days', 'Rent - Apartment', 1500.00, (SELECT id FROM categories WHERE name='Rent' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '22 days', 'Utilities - Electricity', 120.45, (SELECT id FROM categories WHERE name='Utilities' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '20 days', 'Groceries - Whole Foods', 96.72, (SELECT id FROM categories WHERE name='Groceries' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '19 days', 'Subway Pass', 45.00, (SELECT id FROM categories WHERE name='Transportation' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '16 days', 'Movie Night', 28.50, (SELECT id FROM categories WHERE name='Entertainment' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '14 days', 'Groceries - Trader Joes', 64.11, (SELECT id FROM categories WHERE name='Groceries' AND type='expense' LIMIT 1), 'expense', ''),
    (CURRENT_DATE - INTERVAL '13 days', 'Freelance: Dashboard Charts', 600.00, (SELECT id FROM categories WHERE name='Freelance' AND type='income' LIMIT 1), 'income', ''),
	(CURRENT_DATE - INTERVAL '11 days', 'Utilities - Internet', 60.00, (SELECT id FROM categories WHERE name='Utilities' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '8 days', 'Concert Tickets', 140.00, (SELECT id FROM categories WHERE name='Entertainment' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '6 days', 'Groceries - Costco', 132.39, (SELECT id FROM categories WHERE name='Groceries' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '4 days', 'Rideshare', 22.30, (SELECT id FROM categories WHERE name='Transportation' AND type='expense' LIMIT 1), 'expense', ''),
	(CURRENT_DATE - INTERVAL '1 days', 'Dinner Out', 54.80, (SELECT id FROM categories WHERE name='Entertainment' AND type='expense' LIMIT 1), 'expense', '')
	`
	if _, err := tx.Exec(demoTx); err != nil {
		return fmt.Errorf("seeding demo transactions: %w", err)
	}

	// Optional: a couple of demo budgets
	const demoBudgets = `
	INSERT INTO budgets (category_id, amount, period, start_date) VALUES
	((SELECT id FROM categories WHERE name='Groceries' AND type='expense' LIMIT 1), 400.00, 'monthly', date_trunc('month', CURRENT_DATE)::date),
	((SELECT id FROM categories WHERE name='Entertainment' AND type='expense' LIMIT 1), 200.00, 'monthly', date_trunc('month', CURRENT_DATE)::date),
	((SELECT id FROM categories WHERE name='Transportation' AND type='expense' LIMIT 1), 150.00, 'monthly', date_trunc('month', CURRENT_DATE)::date)
	`
	if _, err := tx.Exec(demoBudgets); err != nil {
		return fmt.Errorf("seeding demo budgets: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}


