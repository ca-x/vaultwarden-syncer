package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestExtractEncryptedBackup(t *testing.T) {
	// Create temporary source data
	sourceDir := t.TempDir()
	testFile := filepath.Join(sourceDir, "encrypted_test.txt")
	testContent := []byte("test content for encrypted extraction")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create encrypted backup
	service := NewService(BackupOptions{
		VaultwardenDataPath: sourceDir,
		CompressionLevel:    6,
		Password:            "encryptionpassword",
	})

	ctx := context.Background()
	reader, filename, err := service.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("Failed to create encrypted backup: %v", err)
	}

	// Verify filename is encrypted
	if !strings.HasSuffix(filename, ".enc") {
		t.Errorf("Expected .enc extension for encrypted backup, got: %s", filename)
	}

	// Extract to new directory
	destDir := t.TempDir()
	if err := service.ExtractBackup(ctx, reader, destDir); err != nil {
		t.Fatalf("Failed to extract encrypted backup: %v", err)
	}

	// Verify extracted content
	extractedFile := filepath.Join(destDir, "encrypted_test.txt")
	extractedContent, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if !bytes.Equal(testContent, extractedContent) {
		t.Fatal("Extracted content from encrypted backup doesn't match original")
	}
}

func TestCompressDirectoryWithSubfolders(t *testing.T) {
	// Create complex directory structure
	tempDir := t.TempDir()
	
	// Create main files
	mainFile := filepath.Join(tempDir, "main.txt")
	if err := os.WriteFile(mainFile, []byte("main content"), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, []byte("sub content"), 0644); err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	// Create nested subdirectory
	nestedDir := filepath.Join(subDir, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	nestedFile := filepath.Join(nestedDir, "nested.txt")
	if err := os.WriteFile(nestedFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	// Create backup
	service := NewService(BackupOptions{
		VaultwardenDataPath: tempDir,
		CompressionLevel:    6,
	})

	ctx := context.Background()
	reader, filename, err := service.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify it's a zip file
	if !strings.HasSuffix(filename, ".zip") {
		t.Errorf("Expected .zip extension, got: %s", filename)
	}

	// Read and verify zip content
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read backup data: %v", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

	expectedFiles := map[string]bool{
		"main.txt":            false,
		"subdir/sub.txt":      false,
		"subdir/nested/nested.txt": false,
	}

	for _, file := range zipReader.File {
		if _, exists := expectedFiles[file.Name]; exists {
			expectedFiles[file.Name] = true
		}
	}

	// Verify all expected files are present
	for filename, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not found in archive", filename)
		}
	}
}

func TestDecryptionWithWrongPassword(t *testing.T) {
	// Create test data
	service := NewService(BackupOptions{
		Password: "correctpassword",
	})

	original := []byte("sensitive data to encrypt")
	encrypted, err := service.encryptData(original)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	// Try to decrypt with wrong password
	wrongService := NewService(BackupOptions{
		Password: "wrongpassword",
	})

	_, err = wrongService.DecryptData(encrypted)
	if err == nil {
		t.Fatal("Expected decryption to fail with wrong password")
	}
}

func TestDecryptionWithoutPassword(t *testing.T) {
	service := NewService(BackupOptions{
		// No password set
	})

	fakeEncryptedData := []byte("some fake encrypted data that's longer than 32 bytes to pass initial check")

	_, err := service.DecryptData(fakeEncryptedData)
	if err == nil {
		t.Fatal("Expected decryption to fail when no password is set")
	}

	expectedError := "no password set for decryption"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got: %v", expectedError, err)
	}
}

func TestEncryptionWithEmptyData(t *testing.T) {
	service := NewService(BackupOptions{
		Password: "testpassword",
	})

	encrypted, err := service.encryptData([]byte{})
	if err != nil {
		t.Fatalf("Failed to encrypt empty data: %v", err)
	}

	if len(encrypted) <= 32 {
		t.Fatal("Encrypted data should include salt and be longer than 32 bytes")
	}

	decrypted, err := service.DecryptData(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt empty data: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatal("Decrypted empty data should be empty")
	}
}

func TestCalculateChecksum(t *testing.T) {
	service := NewService(BackupOptions{})

	testData := "test data for checksum calculation"
	reader1 := strings.NewReader(testData)
	reader2 := strings.NewReader(testData)
	reader3 := strings.NewReader("different test data")

	checksum1, err := service.CalculateChecksum(reader1)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	checksum2, err := service.CalculateChecksum(reader2)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	checksum3, err := service.CalculateChecksum(reader3)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	if checksum1 != checksum2 {
		t.Fatal("Identical data should produce identical checksums")
	}

	if checksum1 == checksum3 {
		t.Fatal("Different data should produce different checksums")
	}

	// Verify checksum format (should be hex string)
	if len(checksum1) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("Expected checksum length of 64, got %d", len(checksum1))
	}
}

func TestGetDataInfo(t *testing.T) {
	// Create test directory structure
	tempDir := t.TempDir()
	
	file1 := filepath.Join(tempDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	file2 := filepath.Join(subDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	service := NewService(BackupOptions{
		VaultwardenDataPath: tempDir,
	})

	info, err := service.GetDataInfo()
	if err != nil {
		t.Fatalf("Failed to get data info: %v", err)
	}

	expectedFiles := []string{"file1.txt", "subdir/file2.txt"}
	
	if len(info) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d", len(expectedFiles), len(info))
	}

	for _, expectedFile := range expectedFiles {
		if modTime, exists := info[expectedFile]; !exists {
			t.Errorf("Expected file %s not found in data info", expectedFile)
		} else {
			// Verify the modification time is reasonable (within last minute)
			if time.Since(modTime) > time.Minute {
				t.Errorf("File %s modification time seems too old: %v", expectedFile, modTime)
			}
		}
	}
}

func TestCreateBackupNonExistentPath(t *testing.T) {
	service := NewService(BackupOptions{
		VaultwardenDataPath: "/path/that/does/not/exist",
		CompressionLevel:    6,
	})

	ctx := context.Background()
	_, _, err := service.CreateBackup(ctx)
	if err == nil {
		t.Fatal("Expected backup creation to fail with non-existent path")
	}

	expectedError := "vaultwarden data path does not exist"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got: %v", expectedError, err)
	}
}

func TestExtractBackupInvalidZip(t *testing.T) {
	service := NewService(BackupOptions{})
	ctx := context.Background()
	destDir := t.TempDir()

	invalidData := bytes.NewReader([]byte("this is not a valid zip file"))
	
	err := service.ExtractBackup(ctx, invalidData, destDir)
	if err == nil {
		t.Fatal("Expected extraction to fail with invalid zip data")
	}
}

func TestEncryptLargeData(t *testing.T) {
	service := NewService(BackupOptions{
		Password: "testpassword",
	})

	// Create large test data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := service.encryptData(largeData)
	if err != nil {
		t.Fatalf("Failed to encrypt large data: %v", err)
	}

	decrypted, err := service.DecryptData(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt large data: %v", err)
	}

	if !bytes.Equal(largeData, decrypted) {
		t.Fatal("Large data encryption/decryption failed")
	}
}
