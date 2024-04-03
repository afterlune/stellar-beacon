package repository

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/zlog"
)

func SaveExLog(exLog entity.TExceptionLog) {
	session := ormInit.GetEngine().NewSession()
	defer session.Close()
	session.Begin()
	_, err := session.Prepare().Insert(exLog)
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return
	}
	session.Commit()
}
