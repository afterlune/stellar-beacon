package oss

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	stdErrors "errors"
	"fmt"
	"io"
	"strings"

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

func (s *AliyunStorage) ref(key string) port.ObjectRef {
	return port.ObjectRef{
		Key: key,
		URL: strings.TrimRight(s.publicURL, "/") + "/" + strings.TrimLeft(key, "/"),
	}
}

var _ port.ObjectStorage = (*AliyunStorage)(nil)
