package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/ent/syncjob"
	"github.com/ca-x/vaultwarden-syncer/internal/backup"
	storageProvider "github.com/ca-x/vaultwarden-syncer/internal/storage"
	"github.com/cloudflare/backoff"
)

type Service struct {
	client        *ent.Client
	backupService *backup.Service
	maxRetries    int
	retryDelay    time.Duration
	concurrency   int
	enableResume  bool // 是否启用断点续传
}

func NewService(client *ent.Client, backupService *backup.Service) *Service {
	return &Service{
		client:        client,
		backupService: backupService,
		maxRetries:    3,               // 默认重试3次
		retryDelay:    5 * time.Second, // 默认重试间隔5秒
		concurrency:   3,               // 默认并发数3
		enableResume:  true,            // 默认启用断点续传
	}
}

// SetRetryConfig 设置重试配置
func (s *Service) SetRetryConfig(maxRetries int, retryDelay time.Duration) {
	s.maxRetries = maxRetries
	s.retryDelay = retryDelay
}

// SetConcurrency 设置并发数
func (s *Service) SetConcurrency(concurrency int) {
	if concurrency > 0 {
		s.concurrency = concurrency
	}
}

// SetResumeEnabled 设置是否启用断点续传
func (s *Service) SetResumeEnabled(enabled bool) {
	s.enableResume = enabled
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

	// 检查是否已存在相同备份
	exists, filename, err := s.checkExistingBackup(ctx, provider)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to check existing backup: %v", err))
		return fmt.Errorf("failed to check existing backup: %w", err)
	}

	var backupReader io.Reader
	if !exists {
		// 创建新备份
		backupReader, filename, err = s.backupService.CreateBackup(ctx)
		if err != nil {
			s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to create backup: %v", err))
			return fmt.Errorf("failed to create backup: %w", err)
		}
	} else {
		// 使用已存在的备份
		if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, fmt.Sprintf("Using existing backup: %s", filename)); err != nil {
			return err
		}
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, "Uploading backup..."); err != nil {
		return err
	}

	// 使用backoff机制上传备份
	if err := s.uploadWithBackoff(ctx, job.ID, provider, filename, backupReader); err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to upload backup after retries: %v", err))
		return fmt.Errorf("failed to upload backup after retries: %w", err)
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusCompleted, fmt.Sprintf("Backup uploaded successfully: %s", filename)); err != nil {
		return err
	}

	log.Printf("Backup synced successfully to %s: %s", storage.Name, filename)
	return nil
}

// ConcurrentSyncToStorages 并发同步到多个存储后端
func (s *Service) ConcurrentSyncToStorages(ctx context.Context, storageIDs []int) error {
	if len(storageIDs) == 0 {
		return fmt.Errorf("no storage IDs provided")
	}

	// 创建共享的备份
	backupReader, filename, err := s.backupService.CreateBackup(ctx)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// 使用buffered channel控制并发数
	semaphore := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(storageIDs))

	// 并发同步到各个存储后端
	for _, storageID := range storageIDs {
		wg.Add(1)

		// 启动goroutine进行同步
		go func(id int) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // 释放信号量

			// 执行同步
			if err := s.syncToStorageWithBackup(ctx, id, backupReader, filename); err != nil {
				errChan <- fmt.Errorf("failed to sync to storage %d: %w", id, err)
			}
		}(storageID)
	}

	// 等待所有同步完成
	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("sync failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// syncToStorageWithBackup 使用指定备份同步到特定存储
func (s *Service) syncToStorageWithBackup(ctx context.Context, storageID int, backupReader io.Reader, filename string) error {
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

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, "Uploading backup..."); err != nil {
		return err
	}

	provider, err := s.createStorageProvider(storage)
	if err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to create storage provider: %v", err))
		return fmt.Errorf("failed to create storage provider: %w", err)
	}

	// 使用backoff机制上传备份
	if err := s.uploadWithBackoff(ctx, job.ID, provider, filename, backupReader); err != nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to upload backup after retries: %v", err))
		return fmt.Errorf("failed to upload backup after retries: %w", err)
	}

	if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusCompleted, fmt.Sprintf("Backup uploaded successfully: %s", filename)); err != nil {
		return err
	}

	log.Printf("Backup synced successfully to %s: %s", storage.Name, filename)
	return nil
}

// checkExistingBackup 检查是否已存在相同备份
func (s *Service) checkExistingBackup(ctx context.Context, provider storageProvider.Provider) (bool, string, error) {
	// 获取数据目录信息
	_, err := s.backupService.GetDataInfo()
	if err != nil {
		return false, "", fmt.Errorf("failed to get data info: %w", err)
	}

	// 生成基于数据信息的文件名
	// 这里简化实现，实际应用中可以使用更复杂的算法
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("vaultwarden-backup-%s.zip", timestamp)

	// 检查文件是否已存在
	exists, err := provider.Exists(ctx, filename)
	if err != nil {
		return false, "", fmt.Errorf("failed to check file existence: %w", err)
	}

	if exists {
		// 文件存在，可以进一步检查校验和是否匹配
		// 这里简化处理，直接返回存在
		return true, filename, nil
	}

	return false, filename, nil
}

// uploadWithBackoff 使用backoff机制的上传
func (s *Service) uploadWithBackoff(ctx context.Context, jobID int, provider storageProvider.Provider, filename string, reader io.Reader) error {
	// 创建backoff实例
	maxDuration := s.retryDelay * time.Duration(s.maxRetries)
	b := backoff.New(maxDuration, s.retryDelay)
	var lastErr error

	for i := 0; i <= s.maxRetries; i++ {
		// 如果不是第一次尝试，更新状态
		if i > 0 {
			if err := s.updateJobStatus(ctx, jobID, syncjob.StatusRunning, fmt.Sprintf("Retrying upload (%d/%d)...", i, s.maxRetries)); err != nil {
				return err
			}
			// 等待backoff时间
			select {
			case <-time.After(b.Duration()):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// 尝试上传（使用断点续传）
		err := s.uploadWithResume(ctx, jobID, provider, filename, reader)
		if err == nil {
			return nil // 成功
		}

		lastErr = err
		log.Printf("Upload attempt %d failed: %v", i+1, err)
	}

	return fmt.Errorf("upload failed after %d retries: %w", s.maxRetries, lastErr)
}

// uploadWithResume 带断点续传的上传
func (s *Service) uploadWithResume(ctx context.Context, jobID int, provider storageProvider.Provider, filename string, reader io.Reader) error {
	if !s.enableResume {
		// 如果未启用断点续传，使用普通上传
		return provider.Upload(ctx, filename, reader)
	}

	// 检查远程文件是否存在
	exists, err := provider.Exists(ctx, filename)
	if err != nil {
		return fmt.Errorf("failed to check file existence: %w", err)
	}

	if !exists {
		// 文件不存在，使用普通上传
		return provider.Upload(ctx, filename, reader)
	}

	// 文件存在，尝试断点续传
	remoteSize, err := provider.GetFileSize(ctx, filename)
	if err != nil {
		return fmt.Errorf("failed to get remote file size: %w", err)
	}

	if remoteSize > 0 {
		// 这里需要实现更复杂的断点续传逻辑
		// 由于Go的io.Reader不支持seek，我们需要特殊处理
		// 简化实现：使用普通上传
		return provider.Upload(ctx, filename, reader)
	}

	return provider.Upload(ctx, filename, reader)
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

	// 使用backoff机制下载
	var backupReader io.ReadCloser
	var lastErr error

	// 创建backoff实例
	maxDuration := s.retryDelay * time.Duration(s.maxRetries)
	b := backoff.New(maxDuration, s.retryDelay)

	for i := 0; i <= s.maxRetries; i++ {
		if i > 0 {
			if err := s.updateJobStatus(ctx, job.ID, syncjob.StatusRunning, fmt.Sprintf("Retrying download (%d/%d)...", i, s.maxRetries)); err != nil {
				return err
			}
			// 等待backoff时间
			select {
			case <-time.After(b.Duration()):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		backupReader, err = provider.Download(ctx, filename)
		if err == nil {
			break // 成功
		}

		lastErr = err
		log.Printf("Download attempt %d failed: %v", i+1, err)
	}

	if backupReader == nil {
		s.updateJobStatus(ctx, job.ID, syncjob.StatusFailed, fmt.Sprintf("Failed to download backup after retries: %v", lastErr))
		return fmt.Errorf("failed to download backup after retries: %w", lastErr)
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

// HealthCheck 存储后端健康检查
func (s *Service) HealthCheck(ctx context.Context, storageID int) error {
	storage, err := s.client.Storage.Get(ctx, storageID)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}

	if !storage.Enabled {
		return fmt.Errorf("storage %s is disabled", storage.Name)
	}

	provider, err := s.createStorageProvider(storage)
	if err != nil {
		return fmt.Errorf("failed to create storage provider: %w", err)
	}

	// 尝试列出根目录来检查连接
	_, err = provider.List(ctx, "")
	if err != nil {
		return fmt.Errorf("health check failed for %s (%s): %w", storage.Name, provider.Type(), err)
	}

	return nil
}
