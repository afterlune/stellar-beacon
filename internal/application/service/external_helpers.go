package service

import (
	"context"
	"github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
)

const maxImageUploadBytes int64 = 10 << 20

func uploadMultipart(ctx context.Context, storage port.ObjectStorage, file *multipart.FileHeader, prefix string) (port.ObjectRef, error) {
	if storage == nil {
		return port.ObjectRef{}, errors.Unavailable("storage.upload", nil)
	}
	if file == nil {
		return port.ObjectRef{}, errors.Invalid("storage.upload", "file is required")
	}
	if file.Size > maxImageUploadBytes {
		return port.ObjectRef{}, errors.Invalid("storage.upload", "image file is too large")
	}
	if !isImageUpload(file) {
		return port.ObjectRef{}, errors.Invalid("storage.upload", "only image files are supported")
	}
	key, err := ObjectKey(file.Filename, prefix)
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

func isImageUpload(file *multipart.FileHeader) bool {
	if file == nil {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(file.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "image/") {
		return true
	}
	contentType = strings.ToLower(strings.TrimSpace(mime.TypeByExtension(filepath.Ext(file.Filename))))
	return strings.HasPrefix(contentType, "image/")
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
