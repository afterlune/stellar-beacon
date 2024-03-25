package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
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
