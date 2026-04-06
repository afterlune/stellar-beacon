package repository

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/zlog"
)

func SaveOptLog(optLog entity.TOperationLog) {
	session := ormInit.GetEngine().NewSession()
	defer session.Close()
	// 直接使用引擎插入，不使用事务，减少开销
	// Id为指针类型，默认值为nil，xorm不会尝试插入
	_, err := session.Insert(&optLog)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
}

func SaveExLog(exLog entity.TExceptionLog) {
	session := ormInit.GetEngine().NewSession()
	defer session.Close()
	// 直接使用引擎插入，不使用事务，减少开销
	// Id为指针类型，默认值为nil，xorm不会尝试插入
	_, err := session.Insert(&exLog)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
}
