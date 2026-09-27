package ormInit

import (
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	_ "github.com/lib/pq"
	"log/slog"
	"strings"
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
			slog.Error("initialize database engine failed", "error", err)
			engine = nil
			return
		}
		if environment := strings.ToLower(config.Environment()); environment == "dev" || environment == "integration" || environment == "test" {
			engine.ShowSQL(true)
			engine.Logger().SetLevel(log.LOG_DEBUG)
		} else {
			engine.ShowSQL(false)
			engine.Logger().SetLevel(log.LOG_WARNING)
		}
		// 连接池
		engine.SetMaxIdleConns(10)
		engine.SetMaxOpenConns(100)
		engine.SetConnMaxLifetime(60000)
	})
	return engine
}
