package cmd

import (
	_ "benetnasch/app/infra/config"
	"benetnasch/app/infra/middlewares"
	"benetnasch/app/infra/task"
	"benetnasch/app/infra/tls"
	"benetnasch/app/infra/zlog"
	"benetnasch/route"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "benetnasch",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runServer(); err != nil {
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

func runServer() error {
	banner()

	settings()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.Use(gin.Recovery())
	router.Use(middlewares.Cors())
	router.Use(middlewares.SpiderReject())
	router.Use(middlewares.LoginFilter())
	router.Use(middlewares.AuthorizationFilter())
	router.Use(middlewares.CasbinResourceFilter())
	router.Use(middlewares.AccessLimiter())
	router.Use(middlewares.Log())

	route.RouterSetup(router)

	listener()

	//return router.Run(fmt.Sprintf("%s:%d", viper.GetString("listen.host"), viper.GetInt("listen.port")))
	return runH3(router)
}

func banner() {
	open, err := os.Open("resource/banner.txt")
	if err != nil {
		zlog.Fatal(err.Error())
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
	gin.DisableConsoleColor()

	file, _ := os.Create("resource/log/server.log")

	gin.DefaultWriter = io.MultiWriter(file, os.Stdout)
}

func listener() {
	go task.StatisticsUserArea()
}

func runH3(router *gin.Engine) error {
	cfg := tls.GenerateTLSConfig(0, "")
	h3 := http3.Server{
		Addr:      fmt.Sprintf("%s:%d", viper.GetString("listen.host"), viper.GetInt("listen.port")),
		TLSConfig: cfg,
		QUICConfig: &quic.Config{
			KeepAlivePeriod:    time.Second * 10,
			MaxIdleTimeout:     time.Minute * 30,
			MaxIncomingStreams: 100,
			Allow0RTT:          true,
			EnableDatagrams:    true,
		},
		Handler: router.Handler(),
	}

	h := http.Server{
		Addr:      fmt.Sprintf("%s:%d", viper.GetString("listen.host"), viper.GetInt("listen.port")),
		TLSConfig: cfg,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := h3.SetQUICHeaders(w.Header())
			if err != nil {
				zlog.Error(err.Error())
				return
			}
			router.ServeHTTP(w, r)
		}),
	}
	go func() {
		zlog.Unwrap(h.ListenAndServeTLS("", ""))
	}()
	return h3.ListenAndServe()
}

type H2Handler func(http.ResponseWriter, *http.Request)

func (f H2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}
