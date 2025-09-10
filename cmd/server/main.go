package main

import (
	"context"
	"log"
	"vaultwarden-syncer/ent"
	"vaultwarden-syncer/internal/auth"
	"vaultwarden-syncer/internal/backup"
	"vaultwarden-syncer/internal/config"
	"vaultwarden-syncer/internal/database"
	"vaultwarden-syncer/internal/handler"
	"vaultwarden-syncer/internal/scheduler"
	"vaultwarden-syncer/internal/server"
	"vaultwarden-syncer/internal/service"
	"vaultwarden-syncer/internal/setup"
	"vaultwarden-syncer/internal/sync"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
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
		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server, db *ent.Client, scheduler *scheduler.Service) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					log.Println("Vaultwarden Syncer starting...")
					
					if err := scheduler.Start(ctx); err != nil {
						log.Printf("Failed to start scheduler: %v", err)
					}
					
					go func() {
						if err := srv.Start(); err != nil {
							log.Printf("Server failed to start: %v", err)
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					log.Println("Vaultwarden Syncer stopping...")
					scheduler.Stop()
					if err := database.Close(db); err != nil {
						log.Printf("Error closing database: %v", err)
					}
					return srv.Shutdown()
				},
			})
		}),
	)

	app.Run()
}