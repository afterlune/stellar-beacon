package repository

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"context"
	"testing"
)

func TestLogPersistersRequireInjectedEngine(t *testing.T) {
	ctx := context.Background()
	if err := saveOptLog(nil, ctx, entity.TOperationLog{}); !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("saveOptLog() error = %v, want unavailable", err)
	}
	if err := saveExLog(nil, ctx, entity.TExceptionLog{}); !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("saveExLog() error = %v, want unavailable", err)
	}
}
