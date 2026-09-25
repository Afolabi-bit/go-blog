package media

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2StorageConfig holds the parameters needed to connect to Cloudflare R2.
type R2StorageConfig struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
}

// R2Storage implements Storage by uploading files to Cloudflare R2 via its S3-compatible API.
type R2Storage struct {
	client     *s3.Client
	bucketName string
	publicURL  string
}

// NewR2Storage initializes an S3 client configured for Cloudflare R2.
func NewR2Storage(ctx context.Context, cfg R2StorageConfig) (*R2Storage, error) {
	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for R2: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(r2Endpoint)
	})

	publicURL := strings.TrimRight(cfg.PublicURL, "/")
	if publicURL == "" {
		publicURL = fmt.Sprintf("%s/%s", r2Endpoint, cfg.BucketName)
	}

	return &R2Storage{
		client:     client,
		bucketName: cfg.BucketName,
		publicURL:  publicURL,
	}, nil
}

// Save uploads an object to the configured Cloudflare R2 bucket and returns its public URL.
func (r *R2Storage) Save(ctx context.Context, filename string, reader io.Reader, size int64, mimeType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucketName),
		Key:           aws.String(filename),
		Body:          reader,
		ContentType:   aws.String(mimeType),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object to Cloudflare R2: %w", err)
	}

	return fmt.Sprintf("%s/%s", r.publicURL, filename), nil
}
