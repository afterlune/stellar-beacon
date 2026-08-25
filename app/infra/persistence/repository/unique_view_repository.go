package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
	"context"
)

func ListUniqueViews(ctx context.Context, startTime, endTime string) []model.UniqueViewDTO {
	engine := ormInit.GetEngine().Context(ctx)
	var uqvs []model.UniqueViewDTO
	err := engine.SQL(pgsql.ListUniqueViews, startTime, endTime).Find(&uqvs)
	if err != nil {
		zlog.Error(err.Error())
	}
	return uqvs
}
