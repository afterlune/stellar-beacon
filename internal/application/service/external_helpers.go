package service

import (
	"benetnasch/internal/application/support"
	"benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"context"
	"io"
	"mime/multipart"
	"strings"
)

func uploadMultipart(ctx context.Context, storage port.ObjectStorage, file *multipart.FileHeader, prefix string) (port.ObjectRef, error) {
	if storage == nil {
		return port.ObjectRef{}, errors.Unavailable("storage.upload", nil)
	}
	if file == nil {
		return port.ObjectRef{}, errors.Invalid("storage.upload", "file is required")
	}
	key, err := support.ObjectKey(file.Filename, prefix)
	if err != nil {
		return port.ObjectRef{}, errors.Invalid("storage.upload", err.Error())
	}
	body, err := file.Open()
	if err != nil {
		return port.ObjectRef{}, errors.Unavailable("storage.open", err)
	}
	defer body.Close()
	return storage.Put(ctx, key, body)
}

func uploadNamed(ctx context.Context, storage port.ObjectStorage, reader io.Reader, filename, prefix string) (port.ObjectRef, error) {
	if storage == nil {
		return port.ObjectRef{}, errors.Unavailable("storage.upload", nil)
	}
	if reader == nil || strings.TrimSpace(filename) == "" {
		return port.ObjectRef{}, errors.Invalid("storage.upload", "file content and name are required")
	}
	key := strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(filename, "/")
	return storage.Put(ctx, key, reader)
}
