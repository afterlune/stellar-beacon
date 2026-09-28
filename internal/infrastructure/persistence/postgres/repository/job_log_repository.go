package repository

import (
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"log/slog"
)

func ListJobLogGroups() (s string) {
	engine := ormInit.GetEngine()
	_, err := engine.SQL(pgsql.ListJobLogGroups).Get(&s)
	if err != nil {
		slog.Error("list job log groups failed", "error", err)
	}

	return s
}
