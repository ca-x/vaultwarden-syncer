package database

import (
	"context"
	"fmt"

	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/internal/config"

	_ "github.com/lib-x/entsqlite"
)

func New(cfg *config.Config) (*ent.Client, error) {
	var client *ent.Client
	var err error

	switch cfg.Database.Driver {
	case "sqlite3":
		// 确保数据库文件路径目录存在
		dsn := cfg.Database.DSN
		if dsn == "" {
			dsn = "./data/syncer.db"
		}

		// 使用 entsqlite 驱动名称
		driverName := "sqlite3"

		// 构建完整的DSN，根据SQLite连接参数优化记忆
		// 使用正确的file:前缀格式
		fullDSN := fmt.Sprintf("file:%s?cache=shared&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(10000)", dsn)

		client, err = ent.Open(driverName, fullDSN)
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
