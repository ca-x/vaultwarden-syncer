package cleanup

import (
	"context"
	"log"
	"time"

	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/ent/storage"
	"github.com/ca-x/vaultwarden-syncer/ent/syncjob"
	"github.com/ca-x/vaultwarden-syncer/internal/config"
)

type Service struct {
	client *ent.Client
	config *config.Config
}

func NewService(client *ent.Client, config *config.Config) *Service {
	return &Service{
		client: client,
		config: config,
	}
}

// CleanupOldSyncJobs removes sync job records older than the configured retention period
func (s *Service) CleanupOldSyncJobs(ctx context.Context) error {
	if s.config.Sync.HistoryRetentionDays <= 0 {
		log.Println("History retention is disabled (retention_days <= 0), skipping cleanup")
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -s.config.Sync.HistoryRetentionDays)
	
	log.Printf("Cleaning up sync job records older than %d days (before %s)", 
		s.config.Sync.HistoryRetentionDays, cutoffTime.Format("2006-01-02 15:04:05"))

	deletedCount, err := s.client.SyncJob.
		Delete().
		Where(syncjob.CreatedAtLT(cutoffTime)).
		Exec(ctx)

	if err != nil {
		log.Printf("Failed to cleanup old sync jobs: %v", err)
		return err
	}

	if deletedCount > 0 {
		log.Printf("Successfully cleaned up %d old sync job records", deletedCount)
	} else {
		log.Println("No old sync job records found to cleanup")
	}

	return nil
}

// GetSyncJobStats returns statistics about sync job records
func (s *Service) GetSyncJobStats(ctx context.Context) (map[string]interface{}, error) {
	totalJobs, err := s.client.SyncJob.Query().Count(ctx)
	if err != nil {
		return nil, err
	}

	completedJobs, err := s.client.SyncJob.Query().
		Where(syncjob.StatusEQ(syncjob.StatusCompleted)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	failedJobs, err := s.client.SyncJob.Query().
		Where(syncjob.StatusEQ(syncjob.StatusFailed)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	runningJobs, err := s.client.SyncJob.Query().
		Where(syncjob.StatusEQ(syncjob.StatusRunning)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	pendingJobs, err := s.client.SyncJob.Query().
		Where(syncjob.StatusEQ(syncjob.StatusPending)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// Get oldest and newest records
	var oldestJob, newestJob *ent.SyncJob
	oldestJob, _ = s.client.SyncJob.Query().
		Order(ent.Asc(syncjob.FieldCreatedAt)).
		First(ctx)
	
	newestJob, _ = s.client.SyncJob.Query().
		Order(ent.Desc(syncjob.FieldCreatedAt)).
		First(ctx)

	stats := map[string]interface{}{
		"total_jobs":     totalJobs,
		"completed_jobs": completedJobs,
		"failed_jobs":    failedJobs,
		"running_jobs":   runningJobs,
		"pending_jobs":   pendingJobs,
		"retention_days": s.config.Sync.HistoryRetentionDays,
	}

	if oldestJob != nil {
		stats["oldest_record"] = oldestJob.CreatedAt.Format("2006-01-02 15:04:05")
	}

	if newestJob != nil {
		stats["newest_record"] = newestJob.CreatedAt.Format("2006-01-02 15:04:05")
	}

	return stats, nil
}

// CleanupOldSyncJobsByStorage removes old sync job records for a specific storage
func (s *Service) CleanupOldSyncJobsByStorage(ctx context.Context, storageID int) error {
	if s.config.Sync.HistoryRetentionDays <= 0 {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -s.config.Sync.HistoryRetentionDays)
	
	deletedCount, err := s.client.SyncJob.
		Delete().
		Where(
			syncjob.And(
				syncjob.HasStorageWith(storage.IDEQ(storageID)),
				syncjob.CreatedAtLT(cutoffTime),
			),
		).
		Exec(ctx)

	if err != nil {
		return err
	}

	if deletedCount > 0 {
		log.Printf("Cleaned up %d old sync job records for storage %d", deletedCount, storageID)
	}

	return nil
}