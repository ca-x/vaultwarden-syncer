package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Sync       SyncConfig       `mapstructure:"sync"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Vaultwarden VaultwardenConfig `mapstructure:"vaultwarden"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

type StorageConfig struct {
	WebDAV []WebDAVConfig `mapstructure:"webdav"`
	S3     []S3Config     `mapstructure:"s3"`
}

type WebDAVConfig struct {
	Name     string `mapstructure:"name"`
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (c WebDAVConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

type S3Config struct {
	Name            string `mapstructure:"name"`
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
}

func (c S3Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("access key ID is required")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("secret access key is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	return nil
}

type SyncConfig struct {
	Interval         int    `mapstructure:"interval"`
	CompressionLevel int    `mapstructure:"compression_level"`
	Password         string `mapstructure:"password"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type VaultwardenConfig struct {
	DataPath string `mapstructure:"data_path"`
}

func Load() (*Config, error) {
	viper.SetDefault("server.port", 8181)
	viper.SetDefault("database.driver", "sqlite3")
	viper.SetDefault("database.dsn", "./data/syncer.db")
	viper.SetDefault("sync.interval", 3600)
	viper.SetDefault("sync.compression_level", 6)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "./logs/vaultwarden-syncer.log")
	viper.SetDefault("vaultwarden.data_path", "./data/vaultwarden")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}