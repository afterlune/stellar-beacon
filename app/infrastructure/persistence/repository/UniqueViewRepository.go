package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
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
