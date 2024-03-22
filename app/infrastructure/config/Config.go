package config

import (
	"benetnasch/app/infrastructure/zlog"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// referer host
var Verification string

// email
var SmtpName string
var SmtpEmail string
var SmtpPassword string
var SmtpPort int

// oss
var OssEndPoint string
var OssAccessKeyID string
var OssAccessKeySecret string
var OssBucketName string

var ConfDict map[string]interface{}

func init() {
	env, err := os.Open("resource/config.yaml")
	defer env.Close()
	if err != nil {
		zlog.Error(err.Error())
	}
	envDict := make(map[string]string)
	err = yaml.NewDecoder(env).Decode(&envDict)
	if err != nil {
		zlog.Error(err.Error())
	}
	configEnv, err := os.Open(fmt.Sprintf("resource/config-%s.yaml", envDict["env"]))
	defer configEnv.Close()
	if err != nil {
		zlog.Error(err.Error())
	}
	//ConfDict = make(map[string]interface{})
	err = yaml.NewDecoder(configEnv).Decode(&ConfDict)
	if err != nil {
		zlog.Error(err.Error())
	}
	Verification = ConfDict["verification"].(string)
	smtp()
	oss()
}

type Database struct {
	URL        string
	DriverName string
}

func (base *Database) DataBase() *Database {
	database := ConfDict["database"].(map[string]interface{})
	base.URL = fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=disable", database["driverName"], database["username"], database["password"], database["host"], database["port"], database["baseName"])
	base.DriverName = database["driverName"].(string)
	return base
}

type MeiliSearch struct {
	ApiKey string
	URL    string
}

func (meili *MeiliSearch) MeiliSearch() *MeiliSearch {
	meiliSearch := ConfDict["meiliSearch"].(map[string]interface{})
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
	redisData := ConfDict["redis"].(map[string]interface{})
	redis.Addr = fmt.Sprintf("%s:%d", redisData["host"], redisData["port"])
	redis.Password = redisData["password"].(string)
	redis.DB = redisData["db"].(int)
	return redis
}

type RabbitMQ struct {
	URL string
}

func (rabbit *RabbitMQ) RabbitMQ() *RabbitMQ {
	rabbitData := ConfDict["rabbitmq"].(map[string]interface{})
	rabbit.URL = fmt.Sprintf("amqp://%s:%s@%s:%d/", rabbitData["username"], rabbitData["password"], rabbitData["host"], rabbitData["port"])
	return rabbit
}

func smtp() {
	smtpConf := ConfDict["emailSmtp"].(map[string]interface{})
	SmtpEmail = smtpConf["email"].(string)
	SmtpPassword = smtpConf["password"].(string)
	SmtpPort = smtpConf["port"].(int)
	SmtpName = smtpConf["smtp"].(string)
}

func oss() {
	ossConf := ConfDict["oss"].(map[string]interface{})
	OssBucketName = ossConf["bucketName"].(string)
	OssEndPoint = ossConf["endPoint"].(string)
	OssAccessKeyID = ossConf["accessKeyID"].(string)
	OssAccessKeySecret = ossConf["accessKeySecret"].(string)
}
