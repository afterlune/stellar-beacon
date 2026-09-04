package ormInit

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/infra/config"
	_ "github.com/lib/pq"
	"log/slog"
	"sync"
	"xorm.io/xorm"
	"xorm.io/xorm/log"
)

var engineOnce sync.Once
var engine *xorm.Engine
var err error

// GetEngine 单例初始化一次xorm engine
func GetEngine() *xorm.Engine {
	engineOnce.Do(func() {
		dataBase := new(config.Database).DataBase()
		engine, err = xorm.NewEngine(dataBase.DriverName, dataBase.URL)
		if err != nil {
			slog.Error("initialize database engine failed", "error_code", apperrors.SafeCode(err))
			engine = nil
			return
		}
		xormLogger := newSlogXORMLogger(slog.Default())
		engine.SetLogger(xormLogger)
		engine.ShowSQL(dataBase.ShowSQL)
		if dataBase.ShowSQL {
			xormLogger.SetLevel(log.LOG_INFO)
		} else {
			xormLogger.SetLevel(log.LOG_WARNING)
		}
		// 连接池
		engine.SetMaxIdleConns(10)
		engine.SetMaxOpenConns(100)
		engine.SetConnMaxLifetime(60000)
	})
	return engine
}
