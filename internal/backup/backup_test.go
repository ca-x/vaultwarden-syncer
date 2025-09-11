package backup

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateBackupWithoutPassword(t *testing.T) {
	// Create temporary test data
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := NewService(BackupOptions{
		VaultwardenDataPath: tempDir,
		CompressionLevel:    6,
	})

	ctx := context.Background()
	reader, filename, err := service.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if !strings.Contains(filename, "vaultwarden-backup-") {
		t.Errorf("Unexpected filename format: %s", filename)
	}

	if !strings.HasSuffix(filename, ".zip") {
		t.Errorf("Expected .zip extension, got: %s", filename)
	}

	// Read backup content
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read backup data: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Backup data is empty")
	}
}

func TestCreateBackupWithPassword(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	service := NewService(BackupOptions{
		VaultwardenDataPath: tempDir,
		CompressionLevel:    6,
		Password:            "testpassword",
	})

	ctx := context.Background()
	reader, filename, err := service.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if !strings.HasSuffix(filename, ".enc") {
		t.Errorf("Expected .enc extension, got: %s", filename)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read backup data: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Backup data is empty")
	}
}

func TestEncryptDecryptData(t *testing.T) {
	service := NewService(BackupOptions{
		Password: "testpassword",
	})

	original := []byte("This is test data for encryption")

	encrypted, err := service.encryptData(original)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if bytes.Equal(original, encrypted) {
		t.Fatal("Encrypted data should be different from original")
	}

	decrypted, err := service.DecryptData(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if !bytes.Equal(original, decrypted) {
		t.Fatal("Decrypted data should match original")
	}
}

func TestExtractBackup(t *testing.T) {
	// Create temporary source data
	sourceDir := t.TempDir()
	testFile := filepath.Join(sourceDir, "test.txt")
	testContent := []byte("test content for extraction")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	service := NewService(BackupOptions{
		VaultwardenDataPath: sourceDir,
		CompressionLevel:    6,
	})

	ctx := context.Background()
	reader, _, err := service.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Extract to new directory
	destDir := t.TempDir()
	if err := service.ExtractBackup(ctx, reader, destDir); err != nil {
		t.Fatalf("Failed to extract backup: %v", err)
	}

	// Verify extracted content
	extractedFile := filepath.Join(destDir, "test.txt")
	extractedContent, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if !bytes.Equal(testContent, extractedContent) {
		t.Fatal("Extracted content doesn't match original")
	}
}
