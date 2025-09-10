package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config, err := Load()
	if err != nil {
		// Config file not found is expected in test environment
		if config == nil {
			t.Skip("Config file not found, skipping test")
		}
	}

	// Test default values
	if config.Server.Port != 8181 {
		t.Errorf("Expected default port 8181, got %d", config.Server.Port)
	}

	if config.Database.Driver != "sqlite3" {
		t.Errorf("Expected default database driver 'sqlite3', got %s", config.Database.Driver)
	}

	if config.Database.DSN != "./data/syncer.db" {
		t.Errorf("Expected default DSN './data/syncer.db', got %s", config.Database.DSN)
	}

	if config.Sync.Interval != 3600 {
		t.Errorf("Expected default sync interval 3600, got %d", config.Sync.Interval)
	}

	if config.Sync.CompressionLevel != 6 {
		t.Errorf("Expected default compression level 6, got %d", config.Sync.CompressionLevel)
	}
}

func TestWebDAVConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  WebDAVConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: WebDAVConfig{
				Name:     "test",
				URL:      "https://example.com/webdav",
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: WebDAVConfig{
				URL:      "https://example.com/webdav",
				Username: "user",
				Password: "pass",
			},
			wantErr: true,
		},
		{
			name: "missing URL",
			config: WebDAVConfig{
				Name:     "test",
				Username: "user",
				Password: "pass",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WebDAVConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestS3ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  S3Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: S3Config{
				Name:            "test",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
				Bucket:          "bucket",
			},
			wantErr: false,
		},
		{
			name: "missing access key",
			config: S3Config{
				Name:            "test",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
				Bucket:          "bucket",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("S3Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}