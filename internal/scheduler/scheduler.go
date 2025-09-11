package scheduler

import (
	"context"
	"log"
	"time"
	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/ent/storage"
	"github.com/ca-x/vaultwarden-syncer/internal/config"
	"github.com/ca-x/vaultwarden-syncer/internal/sync"
)

type Service struct {
	client      *ent.Client
	syncService *sync.Service
	config      *config.Config
	ticker      *time.Ticker
	stopChan    chan struct{}
}

func NewService(client *ent.Client, syncService *sync.Service, config *config.Config) *Service {
	return &Service{
		client:      client,
		syncService: syncService,
		config:      config,
		stopChan:    make(chan struct{}),
	}
}

func (s *Service) Start(ctx context.Context) error {
	if s.config.Sync.Interval <= 0 {
		log.Println("Sync scheduler disabled (interval <= 0)")
		return nil
	}

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

	return nil
}

func (s *Service) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
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