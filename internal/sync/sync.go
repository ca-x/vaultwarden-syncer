package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/ent/syncjob"
	"github.com/ca-x/vaultwarden-syncer/internal/backup"
	storageProvider "github.com/ca-x/vaultwarden-syncer/internal/storage"
)

type Service struct {
	client        *ent.Client
	backupService *backup.Service
}

func NewService(client *ent.Client, backupService *backup.Service) *Service {
	return &Service{
		client:        client,
		backupService: backupService,
	}
}

func (s *Service) SyncToStorage(ctx context.Context, storageID int) error {
	storage, err := s.client.Storage.Get(ctx, storageID)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	if !storage.Enabled {
		return fmt.Errorf("storage %s is disabled", storage.Name)
	}

	job, err := s.client.SyncJob.
		Create().
		SetStatus(syncjob.StatusPending).
		SetOperation(syncjob.OperationBackup).
		SetStorageID(storageID).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, "Creating backup..."); err != nil {
		return err
	}

	provider, err := s.createStorageProvider(storage)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to create storage provider: %v", err))
		return fmt.Errorf("failed to create storage provider: %w", err)
	}

	backupReader, filename, err := s.backupService.CreateBackup(ctx)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to create backup: %v", err))
		return fmt.Errorf("failed to create backup: %w", err)
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, "Uploading backup..."); err != nil {
		return err
	}

	if err := provider.Upload(ctx, filename, backupReader); err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to upload backup: %v", err))
		return fmt.Errorf("failed to upload backup: %w", err)
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusCompleted, fmt.Sprintf("Backup uploaded successfully: %s", filename)); err != nil {
		return err
	}

	log.Printf("Backup synced successfully to %s: %s", storage.Name, filename)
	return nil
}

func (s *Service) RestoreFromStorage(ctx context.Context, storageID int, filename, destPath string) error {
	storage, err := s.client.Storage.Get(ctx, storageID)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	job, err := s.client.SyncJob.
		Create().
		SetStatus(syncjob.StatusPending).
		SetOperation(syncjob.OperationRestore).
		SetStorageID(storageID).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, "Downloading backup..."); err != nil {
		return err
	}

	provider, err := s.createStorageProvider(storage)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to create storage provider: %v", err))
		return fmt.Errorf("failed to create storage provider: %w", err)
	}

	backupReader, err := provider.Download(ctx, filename)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to download backup: %v", err))
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer backupReader.Close()

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, "Extracting backup..."); err != nil {
		return err
	}

	if err := s.backupService.ExtractBackup(ctx, backupReader, destPath); err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to extract backup: %v", err))
		return fmt.Errorf("failed to extract backup: %w", err)
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusCompleted, fmt.Sprintf("Backup restored successfully from: %s", filename)); err != nil {
		return err
	}

	log.Printf("Backup restored successfully from %s: %s", storage.Name, filename)
	return nil
}

func (s *Service) updateJobStatus(ctx context.Context, jobID int, status syncjob.Status, message string) error {
	update := s.client.SyncJob.UpdateOneID(jobID).SetStatus(status).SetMessage(message)
	
	if status == syncjob.StatusRunning {
		update = update.SetStartedAt(time.Now())
	} else if status == syncjob.StatusCompleted || status == syncjob.StatusFailed {
		update = update.SetCompletedAt(time.Now())
	}

	_, err := update.Save(ctx)
	return err
}

func (s *Service) createStorageProvider(storage *ent.Storage) (storageProvider.Provider, error) {
	switch storage.Type {
	case "webdav":
		var config storageProvider.WebDAVConfig
		if err := mapConfig(storage.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to parse WebDAV config: %w", err)
		}
		return storageProvider.NewWebDAVProvider(config)
	
	case "s3":
		var config storageProvider.S3Config
		if err := mapConfig(storage.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to parse S3 config: %w", err)
		}
		return storageProvider.NewS3Provider(config)
	
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storage.Type)
	}
}

func mapConfig(source map[string]interface{}, dest interface{}) error {
	// Convert map to JSON and then unmarshal to the destination struct
	jsonData, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := json.Unmarshal(jsonData, dest); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return nil
}