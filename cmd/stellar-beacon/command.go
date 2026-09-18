package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/eternallyzzz/stellar-beacon/internal/bootstrap"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/logging"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/repository"
	appruntime "github.com/eternallyzzz/stellar-beacon/internal/infrastructure/runtime"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/security/tls"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/shared"
	httpapi "github.com/eternallyzzz/stellar-beacon/internal/interfaces/http"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/middleware"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "stellar-beacon",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runServer(); err != nil {
			return err
		}
		return nil
	},
}

func execute() error {
	return rootCmd.Execute()
}

func runServer() error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	if err := shared.ValidateJWTKeys(); err != nil {
		return fmt.Errorf("JWT key initialization failed: %w", err)
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
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()
	runtime, err := bootstrap.Initialize(appCtx)
	if err != nil {
		return fmt.Errorf("application initialization failed: %w", err)
	}
	repository.StartLogQueue(context.Background())
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := repository.StopLogQueue(shutdownCtx); err != nil {
			slog.Error("stop log queue failed", "error", err)
		}
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

	httpapi.RouterSetup(router)

	servers := newServerSet(router)
	serveErrors := make(chan error, 2)
	go func() { serveErrors <- servers.h3.ListenAndServe() }()
	go func() { serveErrors <- servers.h2.ListenAndServeTLS("", "") }()

	appruntime.SetComponent("http", "ready")
	appruntime.SetReady(true)
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	var serveErr error
	select {
	case <-signalCtx.Done():
		slog.Info("shutdown signal received")
	case serveErr = <-serveErrors:
	}

	appruntime.SetReady(false)
	appruntime.SetComponent("http", "stopping")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := servers.h2.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("shutdown HTTP/2 server failed", "error", err)
	}
	if err := servers.h3.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("shutdown HTTP/3 server failed", "error", err)
	}
	if err := runtime.Stop(shutdownCtx); err != nil {
		slog.Error("stop application runtime failed", "error", err)
	}
	if err := repository.StopLogQueue(shutdownCtx); err != nil {
		slog.Error("stop log queue failed", "error", err)
	}
	appruntime.SetComponent("http", "stopped")
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return nil
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

type serverSet struct {
	h2 *http.Server
	h3 *http3.Server
}

func newServerSet(router *gin.Engine) serverSet {
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
	return serverSet{h2: &h, h3: &h3}
}
