package repository

import (
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
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
