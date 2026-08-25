package repository

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"log/slog"
)

func ListMenusByUserInfoId(userInfoId int) []*entity.TMenu {
	engine := ormInit.GetEngine()
	var menu []*entity.TMenu
	err := engine.SQL(pgsql.ListMenusByUserInfoId, userInfoId).Find(&menu)
	if err != nil {
		slog.Error("list menus by user failed", "error", err)
	}
	return menu
}
