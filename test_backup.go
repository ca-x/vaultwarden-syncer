package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ca-x/vaultwarden-syncer/internal/backup"
	"github.com/ca-x/vaultwarden-syncer/internal/config"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// Create logger
	logger, _ := zap.NewDevelopment()

	// Create backup service with config
	backupService := backup.NewService(backup.BackupOptions{
		VaultwardenDataPath: cfg.Vaultwarden.DataPath,
		CompressionLevel:    cfg.Sync.CompressionLevel,
		Password:            cfg.Sync.Password,
		Logger:              logger,
	})

	// Test backup creation
	fmt.Printf("Creating backup from: %s\n", cfg.Vaultwarden.DataPath)
	reader, filename, err := backupService.CreateBackup(context.Background())
	if err != nil {
		fmt.Printf("Failed to create backup: %v\n", err)
		return
	}

	// Save backup to file
	fmt.Printf("Backup created successfully: %s\n", filename)
	outFile, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, reader)
	if err != nil {
		fmt.Printf("Failed to write backup to file: %v\n", err)
		return
	}

	fmt.Printf("Backup saved to: %s\n", filename)
}