package main

import (
	"context"
	"time"

	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/internal/auth"
	"github.com/ca-x/vaultwarden-syncer/internal/backup"
	"github.com/ca-x/vaultwarden-syncer/internal/cleanup"
	"github.com/ca-x/vaultwarden-syncer/internal/config"
	"github.com/ca-x/vaultwarden-syncer/internal/database"
	"github.com/ca-x/vaultwarden-syncer/internal/handler"
	"github.com/ca-x/vaultwarden-syncer/internal/logger"
	"github.com/ca-x/vaultwarden-syncer/internal/notification"
	"github.com/ca-x/vaultwarden-syncer/internal/scheduler"
	"github.com/ca-x/vaultwarden-syncer/internal/server"
	"github.com/ca-x/vaultwarden-syncer/internal/service"
	"github.com/ca-x/vaultwarden-syncer/internal/setup"
	"github.com/ca-x/vaultwarden-syncer/internal/sync"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			func(cfg *config.Config) *zap.Logger {
				logger.InitLogger(cfg.Logging.Level, cfg.Logging.File)
				return logger.GetLogger()
			},
			database.New,
			func(cfg *config.Config) *auth.Service {
				return auth.New(cfg.Auth.JWTSecret)
			},
			func(cfg *config.Config, log *zap.Logger) *backup.Service {
				// Use the Vaultwarden data path from config, with fallback to default
				vaultwardenDataPath := cfg.Vaultwarden.DataPath
				if vaultwardenDataPath == "" {
					vaultwardenDataPath = "./data/vaultwarden"
				}

				return backup.NewService(backup.BackupOptions{
					VaultwardenDataPath: vaultwardenDataPath,
					CompressionLevel:    cfg.Sync.CompressionLevel,
					Password:            cfg.Sync.Password,
					Logger:              log,
				})
			},
			service.NewUserService,
			setup.NewSetupService,
			sync.NewService,
			func(client *ent.Client, cfg *config.Config) *cleanup.Service {
				return cleanup.NewService(client, cfg)
			},
			func(cfg *config.Config) *notification.Service {
				return notification.NewService(&cfg.Notification)
			},
			func(client *ent.Client, syncService *sync.Service, cleanupService *cleanup.Service, cfg *config.Config, notificationService *notification.Service) *scheduler.Service {
				schedulerService := scheduler.NewService(client, syncService, cleanupService, cfg)

				// 设置同步服务的重试配置
				if cfg.Sync.MaxRetries > 0 || cfg.Sync.RetryDelaySeconds > 0 {
					syncService.SetRetryConfig(cfg.Sync.MaxRetries, time.Duration(cfg.Sync.RetryDelaySeconds)*time.Second)
				}

				return schedulerService
			},
			handler.New,
			server.New,
		),
		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server, db *ent.Client, scheduler *scheduler.Service, log *zap.Logger, notificationService *notification.Service) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					log.Info("Vaultwarden Syncer starting...")

					// 启动调度器
					if err := scheduler.Start(ctx); err != nil {
						log.Error("Failed to start scheduler", zap.Error(err))
					}

					// 启动服务器
					go func() {
						if err := srv.Start(); err != nil {
							log.Error("Server failed to start", zap.Error(err))
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					log.Info("Vaultwarden Syncer stopping...")
					scheduler.Stop()
					if err := database.Close(db); err != nil {
						log.Error("Error closing database", zap.Error(err))
					}
					logger.Sync()
					return srv.Shutdown()
				},
			})
		}),
	)

	app.Run()
}
