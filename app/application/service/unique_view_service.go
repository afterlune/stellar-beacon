package service

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/repository"
	"context"
	"time"
)

func listUniqueViews(ctx context.Context) []model.UniqueViewDTO {
	startTime := time.Now().Add(-time.Hour * 24 * 7).Format("2006-01-02")
	endTime := time.Now().Format("2006-01-02")
	data := repository.ListUniqueViews(ctx, startTime, endTime)
	return data
}
