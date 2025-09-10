package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/studio-b12/gowebdav"
)

type WebDAVConfig struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c WebDAVConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

type WebDAVProvider struct {
	config WebDAVConfig
	client *gowebdav.Client
}

func NewWebDAVProvider(config WebDAVConfig) (*WebDAVProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid WebDAV config: %w", err)
	}

	client := gowebdav.NewClient(config.URL, config.Username, config.Password)

	return &WebDAVProvider{
		config: config,
		client: client,
	}, nil
}

func (p *WebDAVProvider) Name() string {
	return p.config.Name
}

func (p *WebDAVProvider) Type() string {
	return "webdav"
}

func (p *WebDAVProvider) Upload(ctx context.Context, path string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	if err := p.client.Write(path, data, 0644); err != nil {
		return fmt.Errorf("failed to upload to WebDAV: %w", err)
	}

	return nil
}

func (p *WebDAVProvider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	data, err := p.client.Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to download from WebDAV: %w", err)
	}

	return io.NopCloser(strings.NewReader(string(data))), nil
}

func (p *WebDAVProvider) Delete(ctx context.Context, path string) error {
	if err := p.client.Remove(path); err != nil {
		return fmt.Errorf("failed to delete from WebDAV: %w", err)
	}
	return nil
}

func (p *WebDAVProvider) List(ctx context.Context, prefix string) ([]string, error) {
	files, err := p.client.ReadDir(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list WebDAV directory: %w", err)
	}

	var result []string
	for _, file := range files {
		if !file.IsDir() {
			result = append(result, file.Name())
		}
	}

	return result, nil
}

func (p *WebDAVProvider) Exists(ctx context.Context, path string) (bool, error) {
	info, err := p.client.Stat(path)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check WebDAV file existence: %w", err)
	}

	return info != nil, nil
}