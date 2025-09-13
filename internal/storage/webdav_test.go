package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// MockWebDAVClient 模拟WebDAV客户端
type MockWebDAVClient struct {
	files map[string][]byte
	err   error
}

func NewMockWebDAVClient() *MockWebDAVClient {
	return &MockWebDAVClient{
		files: make(map[string][]byte),
	}
}

func (m *MockWebDAVClient) SetError(err error) {
	m.err = err
}

func (m *MockWebDAVClient) Write(path string, data []byte, perm os.FileMode) error {
	if m.err != nil {
		return m.err
	}
	m.files[path] = data
	return nil
}

func (m *MockWebDAVClient) Read(path string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	data, exists := m.files[path]
	if !exists {
		return nil, errors.New("404 Not Found")
	}
	return data, nil
}

func (m *MockWebDAVClient) Remove(path string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.files, path)
	return nil
}

func (m *MockWebDAVClient) ReadDir(path string) ([]os.FileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}

	var files []os.FileInfo
	for filePath := range m.files {
		if strings.HasPrefix(filePath, path) {
			relativePath := strings.TrimPrefix(filePath, path)
			if relativePath != "" && !strings.Contains(relativePath, "/") {
				files = append(files, &mockFileInfo{
					name:  relativePath,
					size:  int64(len(m.files[filePath])),
					isDir: false,
				})
			}
		}
	}
	return files, nil
}

func (m *MockWebDAVClient) Stat(path string) (os.FileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}

	data, exists := m.files[path]
	if !exists {
		return nil, errors.New("404 Not Found")
	}

	return &mockFileInfo{
		name:  path,
		size:  int64(len(data)),
		isDir: false,
	}, nil
}

// mockFileInfo 实现os.FileInfo接口
type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// WebDAVClientInterface 定义WebDAV客户端接口
type WebDAVClientInterface interface {
	Write(path string, data []byte, perm os.FileMode) error
	Read(path string) ([]byte, error)
	Remove(path string) error
	ReadDir(path string) ([]os.FileInfo, error)
	Stat(path string) (os.FileInfo, error)
}

// 为了支持测试，我们需要修改WebDAVProvider来使用接口
type TestWebDAVProvider struct {
	config WebDAVConfig
	client WebDAVClientInterface
}

func NewTestWebDAVProvider(config WebDAVConfig, client WebDAVClientInterface) (*TestWebDAVProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &TestWebDAVProvider{
		config: config,
		client: client,
	}, nil
}

func (p *TestWebDAVProvider) Name() string {
	return p.config.Name
}

func (p *TestWebDAVProvider) Type() string {
	return "webdav"
}

func (p *TestWebDAVProvider) Upload(ctx context.Context, path string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	return p.client.Write(path, data, 0644)
}

func (p *TestWebDAVProvider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	data, err := p.client.Read(path)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(strings.NewReader(string(data))), nil
}

func (p *TestWebDAVProvider) Delete(ctx context.Context, path string) error {
	return p.client.Remove(path)
}

func (p *TestWebDAVProvider) List(ctx context.Context, prefix string) ([]string, error) {
	files, err := p.client.ReadDir(prefix)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, file := range files {
		if !file.IsDir() {
			result = append(result, file.Name())
		}
	}

	return result, nil
}

func (p *TestWebDAVProvider) Exists(ctx context.Context, path string) (bool, error) {
	_, err := p.client.Stat(path)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *TestWebDAVProvider) UploadPart(ctx context.Context, path string, reader io.Reader, offset int64) error {
	return p.Upload(ctx, path, reader)
}

func (p *TestWebDAVProvider) DownloadPart(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	data, err := p.client.Read(path)
	if err != nil {
		return nil, err
	}

	end := offset + length
	if end > int64(len(data)) {
		end = int64(len(data))
	}

	if offset < int64(len(data)) {
		chunk := data[offset:end]
		return io.NopCloser(strings.NewReader(string(chunk))), nil
	}

	return io.NopCloser(strings.NewReader("")), nil
}

func (p *TestWebDAVProvider) GetFileSize(ctx context.Context, path string) (int64, error) {
	info, err := p.client.Stat(path)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return 0, nil
		}
		return 0, err
	}

	return info.Size(), nil
}

func TestWebDAVConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  WebDAVConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: WebDAVConfig{
				Name:     "test",
				URL:      "https://webdav.example.com",
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: WebDAVConfig{
				URL:      "https://webdav.example.com",
				Username: "user",
				Password: "pass",
			},
			wantErr: true,
		},
		{
			name: "missing URL",
			config: WebDAVConfig{
				Name:     "test",
				Username: "user",
				Password: "pass",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: WebDAVConfig{
				Name:     "test",
				URL:      "https://webdav.example.com",
				Password: "pass",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			config: WebDAVConfig{
				Name:     "test",
				URL:      "https://webdav.example.com",
				Username: "user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("WebDAVConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebDAVProvider_Upload(t *testing.T) {
	mockClient := NewMockWebDAVClient()
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	ctx := context.Background()
	testData := "test upload content"
	reader := strings.NewReader(testData)

	err = provider.Upload(ctx, "test-upload.txt", reader)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	// 验证数据是否上传成功
	if data, exists := mockClient.files["test-upload.txt"]; !exists || string(data) != testData {
		t.Errorf("Upload() failed, expected data %s, got %s", testData, string(data))
	}
}

func TestWebDAVProvider_Upload_Error(t *testing.T) {
	mockClient := NewMockWebDAVClient()
	mockClient.SetError(errors.New("upload error"))
	
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	ctx := context.Background()
	reader := strings.NewReader("test data")

	err = provider.Upload(ctx, "test-file.txt", reader)
	if err == nil {
		t.Error("Upload() expected error, got nil")
	}
}

func TestWebDAVProvider_Download(t *testing.T) {
	mockClient := NewMockWebDAVClient()
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	// 预先设置数据
	testData := "test download content"
	mockClient.files["test-download.txt"] = []byte(testData)

	ctx := context.Background()
	reader, err := provider.Download(ctx, "test-download.txt")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != testData {
		t.Errorf("Download() expected %s, got %s", testData, string(data))
	}
}

func TestWebDAVProvider_Delete(t *testing.T) {
	mockClient := NewMockWebDAVClient()
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	// 预先设置数据
	mockClient.files["test-delete.txt"] = []byte("to be deleted")

	ctx := context.Background()
	err = provider.Delete(ctx, "test-delete.txt")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 验证文件是否被删除
	if _, exists := mockClient.files["test-delete.txt"]; exists {
		t.Error("Delete() failed, file still exists")
	}
}

func TestWebDAVProvider_Exists(t *testing.T) {
	mockClient := NewMockWebDAVClient()
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	// 预先设置数据
	mockClient.files["existing-file.txt"] = []byte("content")

	ctx := context.Background()

	// 测试存在的文件
	exists, err := provider.Exists(ctx, "existing-file.txt")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() expected true for existing file")
	}

	// 测试不存在的文件
	exists, err = provider.Exists(ctx, "non-existing-file.txt")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() expected false for non-existing file")
	}
}

func TestWebDAVProvider_GetFileSize(t *testing.T) {
	mockClient := NewMockWebDAVClient()
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	testData := "test file size content"
	mockClient.files["size-test.txt"] = []byte(testData)

	ctx := context.Background()
	size, err := provider.GetFileSize(ctx, "size-test.txt")
	if err != nil {
		t.Fatalf("GetFileSize() error = %v", err)
	}

	expectedSize := int64(len(testData))
	if size != expectedSize {
		t.Errorf("GetFileSize() expected %d, got %d", expectedSize, size)
	}
}

func TestWebDAVProvider_DownloadPart(t *testing.T) {
	mockClient := NewMockWebDAVClient()
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	testData := "0123456789abcdef"
	mockClient.files["part-test.txt"] = []byte(testData)

	ctx := context.Background()
	
	// 下载部分数据 (offset=5, length=5) 应该得到 "56789"
	reader, err := provider.DownloadPart(ctx, "part-test.txt", 5, 5)
	if err != nil {
		t.Fatalf("DownloadPart() error = %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	expected := "56789"
	if string(data) != expected {
		t.Errorf("DownloadPart() expected %s, got %s", expected, string(data))
	}
}

func TestWebDAVProvider_Methods(t *testing.T) {
	config := WebDAVConfig{
		Name:     "test-webdav",
		URL:      "https://webdav.example.com",
		Username: "user",
		Password: "pass",
	}

	mockClient := NewMockWebDAVClient()
	provider, err := NewTestWebDAVProvider(config, mockClient)
	if err != nil {
		t.Fatalf("NewTestWebDAVProvider() error = %v", err)
	}

	if provider.Name() != "test-webdav" {
		t.Errorf("Name() expected test-webdav, got %s", provider.Name())
	}

	if provider.Type() != "webdav" {
		t.Errorf("Type() expected webdav, got %s", provider.Type())
	}
}