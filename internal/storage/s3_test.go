package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MockS3Client 模拟S3客户端
type MockS3Client struct {
	objects map[string][]byte
	err     error
}

func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		objects: make(map[string][]byte),
	}
}

func (m *MockS3Client) SetError(err error) {
	m.err = err
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	data, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}

	m.objects[*params.Key] = data
	return &s3.PutObjectOutput{}, nil
}

func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	data, exists := m.objects[*params.Key]
	if !exists {
		return nil, &types.NoSuchKey{}
	}

	rangeHeader := ""
	if params.Range != nil {
		rangeHeader = *params.Range
	}

	if rangeHeader != "" {
		// 解析Range头部
		var start, end int
		if n, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err == nil && n == 2 {
			if start >= 0 && end < len(data) && start <= end {
				data = data[start : end+1]
			}
		}
	}

	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(string(data))),
	}, nil
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	delete(m.objects, *params.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.err != nil {
		return nil, m.err
	}

	var contents []types.Object
	prefix := ""
	if params.Prefix != nil {
		prefix = *params.Prefix
	}

	for key := range m.objects {
		if strings.HasPrefix(key, prefix) {
			contents = append(contents, types.Object{
				Key: aws.String(key),
			})
		}
	}

	return &s3.ListObjectsV2Output{
		Contents: contents,
	}, nil
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	data, exists := m.objects[*params.Key]
	if !exists {
		return nil, &types.NoSuchKey{}
	}

	size := int64(len(data))
	return &s3.HeadObjectOutput{
		ContentLength: &size,
	}, nil
}

// 创建测试用的S3Provider
func createTestS3Provider(mockClient S3ClientInterface) *S3Provider {
	config := S3Config{
		Name:            "test-s3",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Region:          "us-east-1",
		Bucket:          "test-bucket",
	}

	return &S3Provider{
		config: config,
		client: mockClient,
	}
}

func TestS3Config_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  S3Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: S3Config{
				Name:            "test",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
				Bucket:          "bucket",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: S3Config{
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
				Bucket:          "bucket",
			},
			wantErr: true,
		},
		{
			name: "missing access key",
			config: S3Config{
				Name:            "test",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
				Bucket:          "bucket",
			},
			wantErr: true,
		},
		{
			name: "missing secret key",
			config: S3Config{
				Name:        "test",
				AccessKeyID: "key",
				Region:      "us-east-1",
				Bucket:      "bucket",
			},
			wantErr: true,
		},
		{
			name: "missing region",
			config: S3Config{
				Name:            "test",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				Bucket:          "bucket",
			},
			wantErr: true,
		},
		{
			name: "missing bucket",
			config: S3Config{
				Name:            "test",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				Region:          "us-east-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("S3Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestS3Provider_Upload(t *testing.T) {
	mockClient := NewMockS3Client()
	provider := createTestS3Provider(mockClient)

	ctx := context.Background()
	testData := "test data content"
	reader := strings.NewReader(testData)

	err := provider.Upload(ctx, "test-file.txt", reader)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	// 验证数据是否上传成功
	if data, exists := mockClient.objects["test-file.txt"]; !exists || string(data) != testData {
		t.Errorf("Upload() failed, expected data %s, got %s", testData, string(data))
	}
}

func TestS3Provider_Upload_Error(t *testing.T) {
	mockClient := NewMockS3Client()
	mockClient.SetError(errors.New("upload error"))
	provider := createTestS3Provider(mockClient)

	ctx := context.Background()
	reader := strings.NewReader("test data")

	err := provider.Upload(ctx, "test-file.txt", reader)
	if err == nil {
		t.Error("Upload() expected error, got nil")
	}
}

func TestS3Provider_Download(t *testing.T) {
	mockClient := NewMockS3Client()
	provider := createTestS3Provider(mockClient)

	// 预先设置数据
	testData := "test download content"
	mockClient.objects["test-download.txt"] = []byte(testData)

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

func TestS3Provider_Delete(t *testing.T) {
	mockClient := NewMockS3Client()
	provider := createTestS3Provider(mockClient)

	// 预先设置数据
	mockClient.objects["test-delete.txt"] = []byte("to be deleted")

	ctx := context.Background()
	err := provider.Delete(ctx, "test-delete.txt")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 验证文件是否被删除
	if _, exists := mockClient.objects["test-delete.txt"]; exists {
		t.Error("Delete() failed, file still exists")
	}
}

func TestS3Provider_List(t *testing.T) {
	mockClient := NewMockS3Client()
	provider := createTestS3Provider(mockClient)

	// 预先设置数据
	mockClient.objects["backup/file1.txt"] = []byte("content1")
	mockClient.objects["backup/file2.txt"] = []byte("content2")
	mockClient.objects["other/file3.txt"] = []byte("content3")

	ctx := context.Background()
	files, err := provider.List(ctx, "backup/")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	expected := []string{"backup/file1.txt", "backup/file2.txt"}
	if len(files) != len(expected) {
		t.Errorf("List() expected %d files, got %d", len(expected), len(files))
	}

	for _, expectedFile := range expected {
		found := false
		for _, file := range files {
			if file == expectedFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("List() expected file %s not found", expectedFile)
		}
	}
}

func TestS3Provider_Exists(t *testing.T) {
	mockClient := NewMockS3Client()
	provider := createTestS3Provider(mockClient)

	// 预先设置数据
	mockClient.objects["existing-file.txt"] = []byte("content")

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

func TestS3Provider_GetFileSize(t *testing.T) {
	mockClient := NewMockS3Client()
	provider := createTestS3Provider(mockClient)

	testData := "test file size content"
	mockClient.objects["size-test.txt"] = []byte(testData)

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

func TestS3Provider_DownloadPart(t *testing.T) {
	mockClient := NewMockS3Client()
	provider := createTestS3Provider(mockClient)

	testData := "0123456789abcdef"
	mockClient.objects["part-test.txt"] = []byte(testData)

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

func TestS3Provider_Methods(t *testing.T) {
	config := S3Config{
		Name:            "test-s3",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		Bucket:          "bucket",
	}

	provider := &S3Provider{config: config}

	if provider.Name() != "test-s3" {
		t.Errorf("Name() expected test-s3, got %s", provider.Name())
	}

	if provider.Type() != "s3" {
		t.Errorf("Type() expected s3, got %s", provider.Type())
	}
}