package storage

import (
	"context"
	"io"
)

// Provider 定义存储提供者的接口
type Provider interface {
	Name() string
	Type() string
	Upload(ctx context.Context, path string, reader io.Reader) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Exists(ctx context.Context, path string) (bool, error)

	// 新增方法支持断点续传
	UploadPart(ctx context.Context, path string, reader io.Reader, offset int64) error
	DownloadPart(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error)
	GetFileSize(ctx context.Context, path string) (int64, error)
}

// Config 定义存储配置的接口
type Config interface {
	Validate() error
}
