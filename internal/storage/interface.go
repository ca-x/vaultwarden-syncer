package storage

import (
	"context"
	"io"
)

type Provider interface {
	Name() string
	Type() string
	Upload(ctx context.Context, path string, reader io.Reader) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Exists(ctx context.Context, path string) (bool, error)
}

type Config interface {
	Validate() error
}