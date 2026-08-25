package oss

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"fmt"
	"strings"
)

// NewObjectStorage selects the configured object-storage provider. An empty
// provider preserves the existing Aliyun OSS behavior for production configs.
func NewObjectStorage(conf *config.Oss) (port.ObjectStorage, error) {
	if conf == nil {
		return nil, fmt.Errorf("object-storage configuration is nil")
	}
	switch strings.ToLower(strings.TrimSpace(conf.Provider)) {
	case "", "aliyun", "aliyun-oss", "oss":
		return NewAliyunStorage(conf), nil
	case "minio":
		return NewMinioStorage(conf)
	default:
		return nil, fmt.Errorf("unsupported object-storage provider %q", conf.Provider)
	}
}
