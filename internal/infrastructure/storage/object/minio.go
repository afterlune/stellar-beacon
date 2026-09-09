package oss

import (
	"benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/infrastructure/config"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client     *minio.Client
	bucket     string
	publicURL  string
	publicRead bool
}

func NewMinioStorage(conf *config.Oss) (*MinioStorage, error) {
	if conf == nil {
		return nil, fmt.Errorf("MinIO configuration is nil")
	}
	endpoint, secure, err := minioEndpoint(conf.EndPoint)
	if err != nil {
		return nil, err
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(conf.AccessKeyID, conf.AccessKeySecret, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize MinIO client: %w", err)
	}
	return &MinioStorage{
		client:     client,
		bucket:     conf.BucketName,
		publicURL:  conf.PublicURL,
		publicRead: conf.PublicRead,
	}, nil
}

func (s *MinioStorage) Put(ctx context.Context, key string, body io.Reader) (port.ObjectRef, error) {
	if s == nil || s.client == nil {
		return port.ObjectRef{}, errors.Unavailable("minio.put", fmt.Errorf("MinIO client is not configured"))
	}
	if key == "" || body == nil {
		return port.ObjectRef{}, errors.Invalid("minio.put", "object key and body are required")
	}
	if err := s.ensureBucket(ctx); err != nil {
		return port.ObjectRef{}, err
	}
	if _, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{}); err == nil {
		return s.ref(key), nil
	} else if !isMissingObject(err) {
		return port.ObjectRef{}, errors.Unavailable("minio.stat", err)
	}
	if _, err := s.client.PutObject(ctx, s.bucket, key, body, -1, minio.PutObjectOptions{}); err != nil {
		return port.ObjectRef{}, errors.Unavailable("minio.put", err)
	}
	return s.ref(key), nil
}

func (s *MinioStorage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return errors.Unavailable("minio.bucket", err)
	}
	if exists {
		return s.setPublicReadPolicy(ctx)
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		if exists, checkErr := s.client.BucketExists(ctx, s.bucket); checkErr == nil && exists {
			return nil
		}
		return errors.Unavailable("minio.bucket", err)
	}
	return s.setPublicReadPolicy(ctx)
}

func (s *MinioStorage) setPublicReadPolicy(ctx context.Context) error {
	if !s.publicRead {
		return nil
	}
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, s.bucket)
	if err := s.client.SetBucketPolicy(ctx, s.bucket, policy); err != nil {
		return errors.Unavailable("minio.policy", err)
	}
	return nil
}

func (s *MinioStorage) ref(key string) port.ObjectRef {
	return port.ObjectRef{
		Key: key,
		URL: strings.TrimRight(s.publicURL, "/") + "/" + strings.TrimLeft(key, "/"),
	}
}

func minioEndpoint(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, fmt.Errorf("MinIO endpoint is empty")
	}
	if !strings.Contains(raw, "://") {
		return raw, false, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false, fmt.Errorf("parse MinIO endpoint: %w", err)
	}
	if parsed.Scheme == "" {
		return raw, false, nil
	}
	if parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false, fmt.Errorf("MinIO endpoint must be a host with an optional http/https scheme")
	}
	return parsed.Host, strings.EqualFold(parsed.Scheme, "https"), nil
}

func isMissingObject(err error) bool {
	if err == nil {
		return false
	}
	response := minio.ToErrorResponse(err)
	if response.StatusCode == 404 {
		return true
	}
	return response.Code == "NoSuchKey" || response.Code == "NoSuchObject"
}

var _ port.ObjectStorage = (*MinioStorage)(nil)
