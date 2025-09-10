package main

import (
	"context"
	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/internal/auth"
	"github.com/ca-x/vaultwarden-syncer/internal/backup"
	"github.com/ca-x/vaultwarden-syncer/internal/config"
	"github.com/ca-x/vaultwarden-syncer/internal/database"
	"github.com/ca-x/vaultwarden-syncer/internal/handler"
	"github.com/ca-x/vaultwarden-syncer/internal/logger"
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
			func(cfg *config.Config) *backup.Service {
				return backup.NewService(backup.BackupOptions{
					VaultwardenDataPath: "./data/vaultwarden",
					CompressionLevel:    cfg.Sync.CompressionLevel,
					Password:           cfg.Sync.Password,
				})
			},
			service.NewUserService,
			setup.NewSetupService,
			sync.NewService,
			scheduler.NewService,
			handler.New,
			server.New,
		),
		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server, db *ent.Client, scheduler *scheduler.Service, log *zap.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					log.Info("Vaultwarden Syncer starting...")
					
					if err := scheduler.Start(ctx); err != nil {
						log.Error("Failed to start scheduler", zap.Error(err))
					}
					
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