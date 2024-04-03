package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
	"fmt"
)

func ListUniqueViews(startTime, endTime string) []model.UniqueViewDTO {
	s := fmt.Sprintf(pgsql.ListUniqueViews, startTime, endTime)
	engine := ormInit.GetEngine()
	var uqvs []model.UniqueViewDTO
	err := engine.SQL(s).Find(&uqvs)
	if err != nil {
		zlog.Error(err.Error())
	}
	return uqvs
}
