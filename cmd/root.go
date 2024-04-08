package cmd

import (
	"benetnasch/app/infra/middlewares"
	"benetnasch/app/infra/task"
	"benetnasch/app/infra/zlog"
	"benetnasch/route"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "benetnasch",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runServ(); err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		zlog.Fatal(err.Error())
	}
}

func runServ() error {
	// 打印logo
	banner()
	// 设置项
	settings()
	// 创建服务
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	// 配置中间件
	router.Use(gin.Recovery())
	router.Use(middlewares.SpiderReject())
	router.Use(middlewares.Cors())
	router.Use(middlewares.LoginFilter())
	router.Use(middlewares.AuthorizationFilter())
	router.Use(middlewares.CasbinResourceFilter())
	router.Use(middlewares.AccessLimiter())
	router.Use(middlewares.Log())
	// 路由网关
	route.Router(router)
	// 启动消息监听项
	listener()
	// 启动
	return router.Run(fmt.Sprintf("%s:%d", viper.GetString("listen.host"), viper.GetInt("listen.port")))
}

func banner() {
	open, err := os.Open("resource/banner.txt")
	if err != nil {
		zlog.Error(err.Error())
	}
	var data []byte
	buf := make([]byte, 1024)
	for {
		read, err := open.Read(buf)
		if err != nil && err != io.EOF {
			zlog.Error(err.Error())
		}
		if read == 0 {
			break
		}
		data = append(data, buf[:read]...)
	}
	log.Println(string(data))
}

func settings() {
	// 禁用控制台日志颜色
	gin.DisableConsoleColor()
	// 记录到文件
	file, _ := os.Create("resource/log/server.log")
	// 同时将日志写入文件和控制台
	gin.DefaultWriter = io.MultiWriter(file, os.Stdout)
}

func listener() {
	go task.StatisticsUserArea()
}
