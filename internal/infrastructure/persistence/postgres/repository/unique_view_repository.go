package repository

import (
	"benetnasch/internal/infrastructure/persistence/postgres/orm"
	"benetnasch/internal/infrastructure/persistence/postgres/query"
	"benetnasch/internal/interfaces/http/model"
	"context"
	"log/slog"
)

func ListUniqueViews(ctx context.Context, startTime, endTime string) []model.UniqueViewDTO {
	engine := ormInit.GetEngine().Context(ctx)
	var uqvs []model.UniqueViewDTO
	err := engine.SQL(pgsql.ListUniqueViews, startTime, endTime).Find(&uqvs)
	if err != nil {
		slog.Error("list unique views failed", "error", err)
	}
	return uqvs
}
