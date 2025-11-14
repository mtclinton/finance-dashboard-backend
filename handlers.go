package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// healthCheck handles the health check endpoint
func healthCheck(c *gin.Context) {
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "transaction-service",
	})
}

// getTransactions retrieves all transactions with optional Redis caching
func getTransactions(c *gin.Context) {
	ctx := context.Background()

	// Try to get from cache
	if redisClient != nil {
		cached, err := redisClient.Get(ctx, "transactions").Result()
		if err == nil {
			var transactions []Transaction
			if err := json.Unmarshal([]byte(cached), &transactions); err == nil {
				c.JSON(http.StatusOK, transactions)
				return
			}
		}
	}

	// Query database
	query := `
		SELECT t.id, t.date, t.description, t.amount, t.category_id, t.type, t.notes, t.created_at,
		       c.name as category_name, c.color as category_color
		FROM transactions t
		LEFT JOIN categories c ON t.category_id = c.id
		ORDER BY t.date DESC
		LIMIT 100
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// ensure empty array ([]) instead of null when no rows
	transactions := make([]Transaction, 0)
  
	for rows.Next() {
		var t Transaction
		err := rows.Scan(
			&t.ID, &t.Date, &t.Description, &t.Amount, &t.CategoryID, &t.Type, &t.Notes, &t.CreatedAt,
			&t.CategoryName, &t.CategoryColor,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		transactions = append(transactions, t)
	}

	// Cache for 60 seconds
	if redisClient != nil {
		if data, err := json.Marshal(transactions); err == nil {
			redisClient.SetEx(ctx, "transactions", data, 60*time.Second)
		}
	}

	c.JSON(http.StatusOK, transactions)
}

// addTransaction creates a new transaction
func addTransaction(c *gin.Context) {
	var t Transaction
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		INSERT INTO transactions (date, description, amount, category_id, type, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, date, description, amount, category_id, type, notes, created_at
	`

	var result Transaction
	err := db.QueryRow(
		query, t.Date, t.Description, t.Amount, t.CategoryID, t.Type, t.Notes,
	).Scan(
		&result.ID, &result.Date, &result.Description, &result.Amount,
		&result.CategoryID, &result.Type, &result.Notes, &result.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Invalidate cache
	ctx := context.Background()
	if redisClient != nil {
		redisClient.Del(ctx, "transactions")
		redisClient.Del(ctx, "analytics")
	}

	c.JSON(http.StatusCreated, result)
}

// deleteTransaction removes a transaction by ID
func deleteTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}

	_, err = db.Exec("DELETE FROM transactions WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Invalidate cache
	ctx := context.Background()
	if redisClient != nil {
		redisClient.Del(ctx, "transactions")
		redisClient.Del(ctx, "analytics")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted"})
}

// getCategories retrieves all categories
func getCategories(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, type, color, created_at FROM categories ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Color, &cat.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		categories = append(categories, cat)
	}

	c.JSON(http.StatusOK, categories)
}

// getAnalytics retrieves analytics data with optional Redis caching
func getAnalytics(c *gin.Context) {
	ctx := context.Background()

	// Try to get from cache
	if redisClient != nil {
		cached, err := redisClient.Get(ctx, "analytics").Result()
		if err == nil {
			var analytics Analytics
			if err := json.Unmarshal([]byte(cached), &analytics); err == nil {
				c.JSON(http.StatusOK, analytics)
				return
			}
		}
	}

	// Query summary
	summaryQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expenses,
			COUNT(*) as transaction_count
		FROM transactions
		WHERE date >= CURRENT_DATE - INTERVAL '30 days'
	`

	var summary AnalyticsSummary
	err := db.QueryRow(summaryQuery).Scan(
		&summary.TotalIncome, &summary.TotalExpenses, &summary.TransactionCount,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Query by category
	categoryQuery := `
		SELECT c.name, c.color, COALESCE(SUM(t.amount), 0) as total
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.date >= CURRENT_DATE - INTERVAL '30 days' AND t.type = 'expense'
		GROUP BY c.name, c.color
		ORDER BY total DESC
	`

	rows, err := db.Query(categoryQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// ensure empty array ([]) instead of null when no rows
	byCategory := make([]CategoryAnalytics, 0)
  
	for rows.Next() {
		var cat CategoryAnalytics
		err := rows.Scan(&cat.Name, &cat.Color, &cat.Total)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		byCategory = append(byCategory, cat)
	}

	analytics := Analytics{
		Summary:    summary,
		ByCategory: byCategory,
	}

	// Cache for 5 minutes
	if redisClient != nil {
		if data, err := json.Marshal(analytics); err == nil {
			redisClient.SetEx(ctx, "analytics", data, 5*time.Minute)
		}
	}

	c.JSON(http.StatusOK, analytics)
}
