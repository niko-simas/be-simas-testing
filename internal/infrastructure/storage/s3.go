package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
	endpoint   string // optional custom endpoint (for MinIO compatibility)
	region     string
}

type S3Config struct {
	Bucket     string
	Region     string
	AccessKey  string
	SecretKey  string
	Endpoint   string // optional: for MinIO or custom S3-compatible
}

func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	var optFns []func(*config.LoadOptions) error

	optFns = append(optFns,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)

	awsCfg, err := config.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var s3OptFns []func(*s3.Options)
	if cfg.Endpoint != "" {
		s3OptFns = append(s3OptFns, func(o *s3.Options) {
			// MinIO / GCS / custom S3-compatible endpoint
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			// MinIO requires PathStyle
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3OptFns...)
	presigner := s3.NewPresignClient(client)

	return &S3Storage{
		client:    client,
		presigner: presigner,
		bucket:    cfg.Bucket,
		endpoint:  cfg.Endpoint,
		region:    cfg.Region,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("s3 put object: %w", err)
	}

	// Return the S3 URI
	return fmt.Sprintf("s3://%s/%s", s.bucket, key), nil
}

func (s *S3Storage) GeneratePresignedPutURL(ctx context.Context, key string, expiry time.Duration, contentType string) (string, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("s3 presign put: %w", err)
	}
	return req.URL, nil
}

func (s *S3Storage) GeneratePresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("s3 presign get: %w", err)
	}
	return req.URL, nil
}
