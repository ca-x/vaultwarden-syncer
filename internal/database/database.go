package database

import (
	"context"
	"database/sql"
	"fmt"
	"vaultwarden-syncer/internal/config"
	"vaultwarden-syncer/ent"

	_ "github.com/mattn/go-sqlite3"
	"entgo.io/ent/dialect"
)

func New(cfg *config.Config) (*ent.Client, error) {
	var client *ent.Client
	var err error

	switch cfg.Database.Driver {
	case "sqlite3":
		client, err = ent.Open(dialect.SQLite, cfg.Database.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed opening connection to database: %v", err)
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed creating schema resources: %v", err)
	}

	return client, nil
}

func Close(client *ent.Client) error {
	if client != nil {
		return client.Close()
	}
	return nil
}