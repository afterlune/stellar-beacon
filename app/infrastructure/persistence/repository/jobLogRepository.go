package repository

import (
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
)

func ListJobLogGroups() (s string) {
	engine := ormInit.GetEngine()
	_, err := engine.SQL(pgsql.ListJobLogGroups).Get(&s)
	if err != nil {
		zlog.Error(err.Error())
	}

	return s
}
