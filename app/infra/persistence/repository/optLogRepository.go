package repository

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/zlog"
)

func SaveOptLog(optLog entity.TOperationLog) {
	session := ormInit.GetEngine().NewSession()
	defer session.Close()
	session.Begin()
	_, err := session.Prepare().Insert(optLog)
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return
	}
	session.Commit()
}
