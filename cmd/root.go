package cmd

import (
	"benetnasch/app/bootstrap"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/logging"
	"benetnasch/app/infra/middlewares"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/task"
	"benetnasch/app/infra/tls"
	"benetnasch/route"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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

func Execute() error {
	return rootCmd.Execute()
}

func runServer() error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	logDirectory, err := config.ResourcePath("log")
	if err != nil {
		return fmt.Errorf("resolve log directory: %w", err)
	}
	shutdownLogging, err := logging.Init(logging.Config{
		ConsoleLevel: slog.LevelDebug,
		FilePath:     filepath.Join(logDirectory, fmt.Sprintf("error_%s.log", time.Now().Format(time.DateOnly))),
		MaxSize:      50,
		MaxBackups:   5,
		MaxAge:       7,
		Compress:     true,
	})
	if err != nil {
		return fmt.Errorf("initialize logging: %w", err)
	}
	defer func() {
		if err := shutdownLogging(); err != nil {
			slog.Error("close logging failed", "error", err)
		}
	}()
	if err := banner(); err != nil {
		return err
	}

	serverLog, err := settings()
	if err != nil {
		return err
	}
	defer serverLog.Close()
	if err := bootstrap.Initialize(); err != nil {
		return fmt.Errorf("application initialization failed: %w", err)
	}
	repository.StartLogQueue(context.Background())
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := repository.StopLogQueue(shutdownCtx); err != nil {
			slog.Error("stop log queue failed", "error", err)
		}
		cancel()
	}()

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

func banner() error {
	bannerPath, err := config.ResourcePath("banner.txt")
	if err != nil {
		return err
	}
	open, err := os.Open(bannerPath)
	if err != nil {
		return fmt.Errorf("open banner: %w", err)
	}
	defer open.Close()
	var data []byte
	buf := make([]byte, 1024)
	for {
		read, err := open.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read banner: %w", err)
		}
		if read == 0 {
			break
		}
		data = append(data, buf[:read]...)
	}
	log.Println(string(data))
	return nil
}

func settings() (*os.File, error) {
	gin.DisableConsoleColor()

	logPath, err := config.ResourcePath("log/server.log")
	if err != nil {
		return nil, fmt.Errorf("resolve server log path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		return nil, fmt.Errorf("create server log directory: %w", err)
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return nil, fmt.Errorf("create server log: %w", err)
	}

	gin.DefaultWriter = io.MultiWriter(file, os.Stdout)
	return file, nil
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
				slog.Error("set HTTP/3 headers failed", "error", err)
				return
			}
			router.ServeHTTP(w, r)
		}),
	}
	go func() {
		if err := h.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP/3 companion server failed", "error", err)
		}
	}()
	return h3.ListenAndServe()
}

type H2Handler func(http.ResponseWriter, *http.Request)

func (f H2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}
