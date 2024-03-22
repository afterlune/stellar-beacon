package repository

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
	"fmt"
)

func ListMenusByUserInfoId(userInfoId int) []*entity.TMenu {
	s := fmt.Sprintf(pgsql.ListMenusByUserInfoId, userInfoId)
	engine := ormInit.GetEngine()
	var menu []*entity.TMenu
	err := engine.SQL(s).Find(&menu)
	if err != nil {
		zlog.Error(err.Error())
	}
	return menu
}
