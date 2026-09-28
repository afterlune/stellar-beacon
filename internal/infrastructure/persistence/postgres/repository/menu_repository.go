package repository

import (
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
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
