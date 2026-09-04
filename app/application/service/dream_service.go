package service

import (
	"context"
	"strconv"
	"strings"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	defaultDreamPageSize = 20
	maxDreamPageSize     = 100
)

type DreamService interface {
	List(context.Context, DreamQuery) port.ResultVO
}

type DreamQuery struct {
	Current string
	Size    string
}

type MyDreamService struct {
	repository port.DreamRepository
	enabled    bool
}

func NewDreamService(repository port.DreamRepository, enabled bool) (*MyDreamService, error) {
	if enabled && repository == nil {
		return nil, apperrors.Invalid("dream.service.dependencies", "dream repository is required")
	}
	return &MyDreamService{repository: repository, enabled: enabled}, nil
}

func NewDisabledDreamService() *MyDreamService {
	return &MyDreamService{}
}

func (s *MyDreamService) List(ctx context.Context, query DreamQuery) port.ResultVO {
	if s == nil || !s.enabled || s.repository == nil {
		return port.ResultFailWithMessage("梦境暂未公开")
	}
	current, size, err := normalizeDreamQuery(query)
	if err != nil {
		return port.ResultFromError(err)
	}
	entries, count, err := s.repository.ListPublic(ctx, current, size)
	if err != nil {
		return port.ResultFromError(err)
	}
	records := make([]port.DreamDTO, 0, len(entries))
	for _, entry := range entries {
		if entry.Status != port.DreamApproved {
			continue
		}
		records = append(records, port.DreamDTO{
			ID:          entry.ID,
			Title:       entry.Title,
			Content:     entry.Content,
			ImageURL:    entry.ImageURL,
			ImageStatus: string(entry.ImageStatus),
			CreatedAt:   entry.CreatedAt.UTC(),
			UpdatedAt:   entry.UpdatedAt.UTC(),
		})
	}
	return port.ResultOkWithData(port.DreamPageDTO{Records: records, Count: count, HasMore: current*size < count})
}

func normalizeDreamQuery(query DreamQuery) (int, int, error) {
	current := 1
	if value := strings.TrimSpace(query.Current); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return 0, 0, apperrors.Invalid("dream.current", "current must be a positive integer")
		}
		current = parsed
	}
	size := defaultDreamPageSize
	if value := strings.TrimSpace(query.Size); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return 0, 0, apperrors.Invalid("dream.size", "size must be a positive integer")
		}
		size = parsed
	}
	if size > maxDreamPageSize {
		size = maxDreamPageSize
	}
	return current, size, nil
}
