package shared

import (
	apperrors "benetnasch/app/domain/errors"
	"fmt"
	"log/slog"

	"github.com/goccy/go-json"
)

func StructCopy(old, new interface{}) {
	marshal, err := json.Marshal(old)
	if err != nil {
		slog.Error("marshal value for struct copy failed", "error_code", apperrors.SafeCode(err))
		return
	}
	err = json.Unmarshal(marshal, new)
	if err != nil {
		slog.Error("unmarshal value for struct copy failed", "error_code", apperrors.SafeCode(err))
		return
	}
}

// Unmarsh decodes a cached JSON string without panicking on an unexpected
// cache value. Callers can decide whether to fall back to the source of truth.
func Unmarsh(old any, new any) error {
	value, ok := old.(string)
	if !ok {
		return fmt.Errorf("expected cached JSON string, got %T", old)
	}
	return json.Unmarshal([]byte(value), new)
}
