package repository

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
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
