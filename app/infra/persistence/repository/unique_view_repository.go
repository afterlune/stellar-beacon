package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
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
