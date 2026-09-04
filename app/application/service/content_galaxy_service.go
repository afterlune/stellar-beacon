package service

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	defaultGalaxyPageSize = 50
	maxGalaxyPageSize     = 100
)

type ContentGalaxyService interface {
	List(context.Context, ContentGalaxyQuery) port.ResultVO
}

type ContentGalaxyQuery struct {
	Current      string
	Size         string
	LifeStage    string
	UpdatedAfter string
}

type MyContentGalaxyService struct {
	repository port.ContentProjectionRepository
	enabled    bool
}

func NewContentGalaxyService(repository port.ContentProjectionRepository, enabled bool) (*MyContentGalaxyService, error) {
	if enabled && repository == nil {
		return nil, apperrors.Invalid("content.galaxy.dependencies", "content projection repository is required")
	}
	return &MyContentGalaxyService{repository: repository, enabled: enabled}, nil
}

func NewDisabledContentGalaxyService() *MyContentGalaxyService {
	return &MyContentGalaxyService{}
}

func (s *MyContentGalaxyService) List(ctx context.Context, query ContentGalaxyQuery) port.ResultVO {
	if s == nil || !s.enabled || s.repository == nil {
		return port.ResultFailWithMessage("星河暂未公开")
	}
	filter, err := normalizeGalaxyQuery(query)
	if err != nil {
		return port.ResultFromError(err)
	}
	projections, err := s.repository.List(ctx, filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	records := make([]port.ContentGalaxyPointDTO, 0, len(projections))
	var nextSince time.Time
	for _, projection := range projections {
		if projection.IsDeleted || projection.Status != port.ContentProjectionReady || projection.ProjectionX == nil || projection.ProjectionY == nil {
			continue
		}
		if math.IsNaN(*projection.ProjectionX) || math.IsInf(*projection.ProjectionX, 0) || math.IsNaN(*projection.ProjectionY) || math.IsInf(*projection.ProjectionY, 0) {
			return port.ResultFromError(apperrors.Unavailable("content.galaxy.data", nil))
		}
		stage, stageErr := port.NormalizeContentLifeStage(string(projection.LifeStage))
		if stageErr != nil {
			return port.ResultFromError(apperrors.Unavailable("content.galaxy.data", stageErr))
		}
		records = append(records, port.ContentGalaxyPointDTO{
			ArticleID: projection.ArticleID,
			LifeStage: string(stage),
			X:         *projection.ProjectionX,
			Y:         *projection.ProjectionY,
			UpdatedAt: projection.UpdatedAt.UTC(),
		})
		if projection.UpdatedAt.After(nextSince) {
			nextSince = projection.UpdatedAt.UTC()
		}
	}
	page := port.ContentGalaxyPageDTO{
		Records: records,
		Count:   len(records),
		HasMore: len(projections) >= filter.Size,
	}
	if !nextSince.IsZero() {
		page.NextSince = nextSince.Format(time.RFC3339Nano)
	}
	return port.ResultOkWithData(page)
}

func normalizeGalaxyQuery(query ContentGalaxyQuery) (port.ContentProjectionFilter, error) {
	current := 1
	if strings.TrimSpace(query.Current) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(query.Current))
		if err != nil || parsed <= 0 {
			return port.ContentProjectionFilter{}, apperrors.Invalid("content.galaxy.current", "current must be a positive integer")
		}
		current = parsed
	}
	size := defaultGalaxyPageSize
	if strings.TrimSpace(query.Size) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(query.Size))
		if err != nil || parsed <= 0 {
			return port.ContentProjectionFilter{}, apperrors.Invalid("content.galaxy.size", "size must be a positive integer")
		}
		size = parsed
	}
	if size > maxGalaxyPageSize {
		size = maxGalaxyPageSize
	}
	stage := port.ContentLifeStage(strings.TrimSpace(strings.ToLower(query.LifeStage)))
	if stage != "" {
		if _, err := port.NormalizeContentLifeStage(string(stage)); err != nil {
			return port.ContentProjectionFilter{}, apperrors.Invalid("content.galaxy.stage", "stage is invalid")
		}
	}
	var updatedAfter time.Time
	if strings.TrimSpace(query.UpdatedAfter) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(query.UpdatedAfter))
		if err != nil {
			return port.ContentProjectionFilter{}, apperrors.Invalid("content.galaxy.since", "since must be RFC3339")
		}
		updatedAfter = parsed.UTC()
	}
	return port.ContentProjectionFilter{
		Current:      current,
		Size:         size,
		LifeStage:    stage,
		Status:       port.ContentProjectionReady,
		UpdatedAfter: updatedAfter,
	}, nil
}
