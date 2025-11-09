package main

// Transaction represents a financial transaction
type Transaction struct {
	ID            int     `json:"id"`
	Date          string  `json:"date"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
	CategoryID    *int    `json:"category_id"`
	Type          string  `json:"type"`
	Notes         *string `json:"notes"`
	CreatedAt     string  `json:"created_at"`
	CategoryName  *string `json:"category_name"`
	CategoryColor *string `json:"category_color"`
}

// Category represents a transaction category
type Category struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Color     string `json:"color"`
	CreatedAt string `json:"created_at"`
}

// AnalyticsSummary contains summary statistics for analytics
type AnalyticsSummary struct {
	TotalIncome      float64 `json:"total_income"`
	TotalExpenses    float64 `json:"total_expenses"`
	TransactionCount int     `json:"transaction_count"`
}

// CategoryAnalytics contains analytics data for a specific category
type CategoryAnalytics struct {
	Name  string  `json:"name"`
	Color string  `json:"color"`
	Total float64 `json:"total"`
}

// Analytics contains all analytics data
type Analytics struct {
	Summary    AnalyticsSummary    `json:"summary"`
	ByCategory []CategoryAnalytics `json:"byCategory"`
}
