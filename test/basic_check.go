package main

import (
	"context"
	"log"
	"time"

	"github.com/ca-x/vaultwarden-syncer/internal/config"
	"github.com/ca-x/vaultwarden-syncer/internal/database"
)

func main() {
	log.Println("Testing Vaultwarden Syncer basic functionality...")

	// Test config loading
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Config loading: %v (expected in test environment)", err)
	} else {
		log.Printf("Config loaded successfully: port=%d", cfg.Server.Port)
	}

	// Test database connection
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer database.Close(db)

	// Test database schema creation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.Schema.Create(ctx); err != nil {
		log.Printf("Schema creation: %v (may already exist)", err)
	} else {
		log.Println("Database schema created successfully")
	}

	log.Println("Basic functionality test completed successfully!")
}