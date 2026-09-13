package oss

import (
	"context"
	stdErrors "errors"
	"fmt"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"io"
	"strings"
	"time"

	aliyunoss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

type AliyunStorage struct {
	client    *aliyunoss.Client
	bucket    string
	publicURL string
}

func NewAliyunStorage(conf *config.Oss) *AliyunStorage {
	provider := credentials.NewStaticCredentialsProvider(conf.AccessKeyID, conf.AccessKeySecret)
	options := aliyunoss.LoadDefaultConfig().
		WithCredentialsProvider(provider).
		WithRegion(conf.Region).
		WithEndpoint(conf.EndPoint)
	return &AliyunStorage{
		client:    aliyunoss.NewClient(options),
		bucket:    conf.BucketName,
		publicURL: conf.PublicURL,
	}
}

func (s *AliyunStorage) Put(ctx context.Context, key string, body io.Reader) (port.ObjectRef, error) {
	if s == nil || s.client == nil {
		return port.ObjectRef{}, errors.Unavailable("oss.put", fmt.Errorf("OSS client is not configured"))
	}
	if key == "" || body == nil {
		return port.ObjectRef{}, errors.Invalid("oss.put", "object key and body are required")
	}
	if _, err := s.client.HeadObject(ctx, &aliyunoss.HeadObjectRequest{
		Bucket: aliyunoss.Ptr(s.bucket),
		Key:    aliyunoss.Ptr(key),
	}); err == nil {
		return s.ref(key), nil
	} else {
		var serviceErr *aliyunoss.ServiceError
		if !stdErrors.As(err, &serviceErr) || serviceErr.HttpStatusCode() != 404 {
			return port.ObjectRef{}, errors.Unavailable("oss.head", err)
		}
	}
	_, err := s.client.PutObject(ctx, &aliyunoss.PutObjectRequest{
		Bucket:          aliyunoss.Ptr(s.bucket),
		Key:             aliyunoss.Ptr(key),
		Body:            body,
		ForbidOverwrite: aliyunoss.Ptr("true"),
	})
	if err != nil {
		return port.ObjectRef{}, errors.Unavailable("oss.put", err)
	}
	return s.ref(key), nil
}

func (s *AliyunStorage) List(ctx context.Context, prefix string) ([]port.ObjectInfo, error) {
	if s == nil || s.client == nil {
		return nil, errors.Unavailable("oss.list", fmt.Errorf("OSS client is not configured"))
	}
	result, err := s.client.ListObjects(ctx, &aliyunoss.ListObjectsRequest{
		Bucket:  aliyunoss.Ptr(s.bucket),
		Prefix:  aliyunoss.Ptr(prefix),
		MaxKeys: 1000,
	})
	if err != nil {
		return nil, errors.Unavailable("oss.list", err)
	}
	objects := make([]port.ObjectInfo, 0, len(result.Contents))
	for _, object := range result.Contents {
		if object.Key == nil || *object.Key == "" {
			continue
		}
		lastModified := time.Time{}
		if object.LastModified != nil {
			lastModified = *object.LastModified
		}
		objects = append(objects, port.ObjectInfo{
			Key:          *object.Key,
			URL:          s.ref(*object.Key).URL,
			Size:         object.Size,
			LastModified: lastModified,
		})
	}
	return objects, nil
}

func (s *AliyunStorage) Delete(ctx context.Context, keys []string) error {
	if s == nil || s.client == nil {
		return errors.Unavailable("oss.delete", fmt.Errorf("OSS client is not configured"))
	}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, err := s.client.DeleteObject(ctx, &aliyunoss.DeleteObjectRequest{
			Bucket: aliyunoss.Ptr(s.bucket),
			Key:    aliyunoss.Ptr(key),
		}); err != nil {
			return errors.Unavailable("oss.delete", err)
		}
	}
	return nil
}

func (s *AliyunStorage) ref(key string) port.ObjectRef {
	return port.ObjectRef{
		Key: key,
		URL: strings.TrimRight(s.publicURL, "/") + "/" + strings.TrimLeft(key, "/"),
	}
}

var _ port.ObjectStorage = (*AliyunStorage)(nil)
var _ port.MediaStorage = (*AliyunStorage)(nil)
