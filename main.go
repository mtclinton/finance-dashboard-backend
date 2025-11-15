package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Check for migrate command
	migrateCmd := flag.Bool("migrate", false, "Run database migration and seed data")
	seedDemoCmd := flag.Bool("seed-demo", false, "Seed demo transactions and budgets (idempotent)")
	flag.Parse()

	if *migrateCmd {
		if err := setupDatabase(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migration completed successfully")
		os.Exit(0)
	}
	if *seedDemoCmd {
		if err := initDB(); err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		defer db.Close()
		if err := seedDemoData(db); err != nil {
			log.Fatalf("Seeding demo data failed: %v", err)
		}
		log.Println("Demo data seeded")
		os.Exit(0)
	}
	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	if err := initRedis(); err != nil {
		log.Printf("Warning: Failed to initialize Redis: %v", err)
		log.Println("Continuing without Redis cache...")
		redisClient = nil
	}

	// Setup Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Routes
	r.GET("/health", healthCheck)
	r.GET("/api/transactions", getTransactions)
	r.POST("/api/transactions", addTransaction)
	r.DELETE("/api/transactions/:id", deleteTransaction)
	r.GET("/api/categories", getCategories)
	r.GET("/api/analytics", getAnalytics)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
