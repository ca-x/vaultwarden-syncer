package scheduler

import (
	"context"
	"log"
	"time"
	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/ent/storage"
	"github.com/ca-x/vaultwarden-syncer/internal/cleanup"
	"github.com/ca-x/vaultwarden-syncer/internal/config"
	"github.com/ca-x/vaultwarden-syncer/internal/sync"
)

type Service struct {
	client         *ent.Client
	syncService    *sync.Service
	cleanupService *cleanup.Service
	config         *config.Config
	ticker         *time.Ticker
	cleanupTicker  *time.Ticker
	stopChan       chan struct{}
}

func NewService(client *ent.Client, syncService *sync.Service, cleanupService *cleanup.Service, config *config.Config) *Service {
	return &Service{
		client:         client,
		syncService:    syncService,
		cleanupService: cleanupService,
		config:         config,
		stopChan:       make(chan struct{}),
	}
}

func (s *Service) Start(ctx context.Context) error {
	// Start sync scheduler if enabled
	if s.config.Sync.Interval > 0 {
		interval := time.Duration(s.config.Sync.Interval) * time.Second
		s.ticker = time.NewTicker(interval)

		go func() {
			log.Printf("Sync scheduler started with interval: %v", interval)
			
			for {
				select {
				case <-s.ticker.C:
					if err := s.runSync(ctx); err != nil {
						log.Printf("Scheduled sync failed: %v", err)
					}
				case <-s.stopChan:
					log.Println("Sync scheduler stopped")
					return
				}
			}
		}()
	} else {
		log.Println("Sync scheduler disabled (interval <= 0)")
	}

	// Start cleanup scheduler if history retention is enabled
	if s.config.Sync.HistoryRetentionDays > 0 {
		// Run cleanup daily at 2 AM
		cleanupInterval := 24 * time.Hour
		s.cleanupTicker = time.NewTicker(cleanupInterval)

		go func() {
			log.Printf("Cleanup scheduler started with daily interval")
			
			// Run initial cleanup after 1 minute
			time.Sleep(1 * time.Minute)
			if err := s.runCleanup(ctx); err != nil {
				log.Printf("Initial cleanup failed: %v", err)
			}
			
			for {
				select {
				case <-s.cleanupTicker.C:
					if err := s.runCleanup(ctx); err != nil {
						log.Printf("Scheduled cleanup failed: %v", err)
					}
				case <-s.stopChan:
					log.Println("Cleanup scheduler stopped")
					return
				}
			}
		}()
	} else {
		log.Println("Cleanup scheduler disabled (history_retention_days <= 0)")
	}

	return nil
}

func (s *Service) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}
	close(s.stopChan)
}

func (s *Service) runSync(ctx context.Context) error {
	storages, err := s.client.Storage.
		Query().
		Where(storage.Enabled(true)).
		All(ctx)

	if err != nil {
		return err
	}

	if len(storages) == 0 {
		log.Println("No enabled storage backends found for sync")
		return nil
	}

	log.Printf("Starting scheduled sync to %d storage backends", len(storages))

	for _, st := range storages {
		if err := s.syncService.SyncToStorage(ctx, st.ID); err != nil {
			log.Printf("Failed to sync to storage %s: %v", st.Name, err)
			continue
		}
	}

	log.Println("Scheduled sync completed")
	return nil
}

func (s *Service) RunSyncNow(ctx context.Context) error {
	log.Println("Manual sync triggered")
	return s.runSync(ctx)
}

func (s *Service) runCleanup(ctx context.Context) error {
	log.Println("Starting scheduled cleanup of old sync job records")
	
	if err := s.cleanupService.CleanupOldSyncJobs(ctx); err != nil {
		return err
	}
	
	log.Println("Scheduled sync job cleanup completed")
	return nil
}

func (s *Service) RunCleanupNow(ctx context.Context) error {
	log.Println("Manual cleanup triggered")
	return s.runCleanup(ctx)
}