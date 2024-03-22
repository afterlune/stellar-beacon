package service

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/repository"
	"time"
)

func listUniqueViews() []model.UniqueViewDTO {
	startTime := time.Now().Add(-time.Hour * 24 * 7).Format("2006-01-02")
	endTime := time.Now().Format("2006-01-02")
	data := repository.ListUniqueViews(startTime, endTime)
	return data
}
