package config

import (
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/zlog"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	xormadapter "github.com/casbin/xorm-adapter/v2"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"log"
	"os"
	"sync"
)

var (
	// Verification referer host
	Verification string

	workDir, _ = os.Getwd()
	configPath = "/resource/config.yaml"
)

func init() {
	viper.SetConfigFile(workDir + configPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln("读取配置文件失败：", err)
	}

	Verification = viper.GetString("verification")
}

type Database struct {
	URL        string
	DriverName string
}

func (base *Database) DataBase() *Database {
	database := viper.GetStringMap("database")
	base.URL = fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=disable", database["driverName"], database["username"], database["password"], database["host"], database["port"], database["baseName"])
	base.DriverName = database["driverName"].(string)
	return base
}

type MeiliSearch struct {
	ApiKey string
	URL    string
}

func (meili *MeiliSearch) MeiliSearch() *MeiliSearch {
	meiliSearch := viper.GetStringMap("meiliSearch")
	meili.ApiKey = meiliSearch["apiKey"].(string)
	meili.URL = fmt.Sprintf("http://%s:%d", meiliSearch["host"], meiliSearch["port"])
	return meili
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

func (redis *Redis) Redis() *Redis {
	redisData := viper.GetStringMap("redis")
	redis.Addr = fmt.Sprintf("%s:%d", redisData["host"], redisData["port"])
	redis.Password = redisData["password"].(string)
	redis.DB = redisData["db"].(int)
	return redis
}

type Email struct {
	EmailAccount string
	Password     string
	SmtpPort     int
	SmtpName     string
}

func (e *Email) Email() *Email {
	smtpConf := viper.GetStringMap("email")
	e.EmailAccount = smtpConf["email"].(string)
	e.Password = smtpConf["password"].(string)
	e.SmtpPort = smtpConf["port"].(int)
	e.SmtpName = smtpConf["smtp"].(string)
	return e
}

type Oss struct {
	BucketName      string
	EndPoint        string
	AccessKeyID     string
	AccessKeySecret string
}

func (o *Oss) Oss() *Oss {
	ossConf := viper.GetStringMap("oss")
	o.BucketName = ossConf["bucketName"].(string)
	o.EndPoint = ossConf["endPoint"].(string)
	o.AccessKeyID = ossConf["accessKeyID"].(string)
	o.AccessKeySecret = ossConf["accessKeySecret"].(string)
	return o
}

var (
	adapterOnce  sync.Once
	enforcerOnce sync.Once
	xormAdapter  *xormadapter.Adapter
	enforcer     *casbin.Enforcer
)

const (
	text = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`
)

// casbinAdapter 外部存储，如果需要的话
func casbinAdapter() *xormadapter.Adapter {
	adapterOnce.Do(func() {
		a, err := xormadapter.NewAdapterByEngine(ormInit.GetEngine())
		if err != nil {
			zlog.Fatal(err.Error())
		}
		xormAdapter = a
	})
	return xormAdapter
}

func CasbinEnforcer() *casbin.Enforcer {
	enforcerOnce.Do(func() {
		m, err := model.NewModelFromString(text)
		if err != nil {
			zlog.Fatal(err.Error())
		}

		e, err := casbin.NewEnforcer(m)
		if err != nil {
			zlog.Fatal(err.Error())
		}
		ok, err := e.AddPolicies([][]string{
			{"admin", "/admin/*", "GET"},
			{"admin", "/admin/*", "PUT"},
			{"admin", "/admin/*", "POST"},
			{"admin", "/admin/*", "DELETE"},
		})
		if !ok || err != nil {
			zlog.Fatal("策略未成功添加")
		}
		enforcer = e
	})
	err := enforcer.LoadPolicy()
	if err != nil {
		zlog.Fatal("策略未成功加载", zap.Error(err))
	}
	return enforcer
}
