package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Config struct {
	Name            string `json:"name"`
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
}

func (c S3Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("access key ID is required")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("secret access key is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	return nil
}

type S3Provider struct {
	config S3Config
	client *s3.Client
}

func NewS3Provider(config S3Config) (*S3Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid S3 config: %w", err)
	}

	cfg, err := awsConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Provider{
		config: config,
		client: client,
	}, nil
}

func awsConfig(c S3Config) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(c.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.AccessKeyID,
			c.SecretAccessKey,
			"",
		)),
	)

	if err != nil {
		return aws.Config{}, err
	}

	if c.Endpoint != "" {
		cfg.BaseEndpoint = aws.String(c.Endpoint)
	}

	return cfg, nil
}

func (p *S3Provider) Name() string {
	return p.config.Name
}

func (p *S3Provider) Type() string {
	return "s3"
}

func (p *S3Provider) Upload(ctx context.Context, path string, reader io.Reader) error {
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(path),
		Body:   reader,
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func (p *S3Provider) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	result, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, nil
}

func (p *S3Provider) Delete(ctx context.Context, path string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

func (p *S3Provider) List(ctx context.Context, prefix string) ([]string, error) {
	result, err := p.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.config.Bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	var files []string
	for _, obj := range result.Contents {
		if obj.Key != nil {
			files = append(files, *obj.Key)
		}
	}

	return files, nil
}

func (p *S3Provider) Exists(ctx context.Context, path string) (bool, error) {
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		// Check if it's a "not found" error
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check S3 object existence: %w", err)
	}

	return true, nil
}

// UploadPart 上传文件的一部分（支持断点续传）
func (p *S3Provider) UploadPart(ctx context.Context, path string, reader io.Reader, offset int64) error {
	// S3的分段上传需要更复杂的实现，这里简化为完整上传
	// 在生产环境中，应该使用S3的multipart upload功能
	return p.Upload(ctx, path, reader)
}

// DownloadPart 下载文件的一部分（支持断点续传）
func (p *S3Provider) DownloadPart(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	// 使用Range参数下载文件的一部分
	rangeHeader := fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)
	
	result, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(path),
		Range:  aws.String(rangeHeader),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to download part from S3: %w", err)
	}

	return result.Body, nil
}

// GetFileSize 获取文件大小
func (p *S3Provider) GetFileSize(ctx context.Context, path string) (int64, error) {
	result, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		// Check if it's a "not found" error
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get S3 object size: %w", err)
	}

	if result.ContentLength != nil {
		return *result.ContentLength, nil
	}

	return 0, nil
}