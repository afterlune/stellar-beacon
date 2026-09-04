package cmd

import (
	"benetnasch/app/bootstrap"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/infra/config"
	"benetnasch/app/infra/logging"
	"benetnasch/app/infra/middlewares"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/task"
	"benetnasch/app/infra/tls"
	"benetnasch/route"
	"context"
	"errors"
	"fmt"
	"io"
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
	Use:   "benetnasch",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runServer(cmd.Context()); err != nil {
			return err
		}
		return nil
	},
}

func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return rootCmd.ExecuteContext(ctx)
}

func runServer(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
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
			slog.Error("close logging failed", "error_code", apperrors.SafeCode(err))
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
	runtime, err := bootstrap.InitializeRuntime()
	if err != nil {
		return fmt.Errorf("application initialization failed: %w", err)
	}
	appCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	repository.StartLogQueue(appCtx, ormInit.GetEngine())
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := repository.StopLogQueue(shutdownCtx); err != nil {
			slog.Error("stop log queue failed", "error_code", apperrors.SafeCode(err))
		}
		cancel()
	}()
	workers, err := listener(appCtx, runtime.ArticleIndexWorker, runtime.ContentUnderstandingWorker, runtime.AgentBehaviorWorker, runtime.ContentProjectionWorker, runtime.DreamWorker, runtime.DreamImageWorker, runtime.TimeCapsuleWorker)
	if err != nil {
		return fmt.Errorf("start background workers: %w", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := workers.Stop(shutdownCtx); err != nil {
			slog.Error("stop background workers failed", "error_code", apperrors.SafeCode(err))
		}
		shutdownCancel()
	}()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	trustedProxies, err := config.TrustedProxies()
	if err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}
	// Gin trusts every proxy by default. Keep forwarded client-IP headers
	// disabled unless the deployment explicitly identifies its ingress CIDRs.
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}
	slog.Info("trusted proxy client IP parsing configured", "count", len(trustedProxies))

	router.Use(gin.Recovery())
	router.Use(middlewares.Cors())
	router.Use(middlewares.SpiderReject())
	router.Use(middlewares.LoginFilter())
	router.Use(middlewares.AuthorizationFilter())
	router.Use(middlewares.CasbinResourceFilter())
	router.Use(middlewares.AccessLimiter())
	router.Use(middlewares.Log())

	route.RouterSetup(router)

	//return router.Run(fmt.Sprintf("%s:%d", viper.GetString("listen.host"), viper.GetInt("listen.port")))
	return runH3(appCtx, router)
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
	if _, err := fmt.Fprintln(os.Stdout, string(data)); err != nil {
		return fmt.Errorf("write banner: %w", err)
	}
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

func listener(ctx context.Context, optionalWorkers ...task.Worker) (*task.Supervisor, error) {
	supervisor := task.NewSupervisor(ctx)
	if err := supervisor.Add("statistics_user_area", task.StatisticsUserArea); err != nil {
		return nil, err
	}
	workerNames := []string{"article_index", "content_understanding", "agent_behavior", "content_projection", "agent_dream", "agent_dream_image", "time_capsule"}
	for index, worker := range optionalWorkers {
		if worker == nil {
			continue
		}
		name := "optional_worker"
		if index < len(workerNames) {
			name = workerNames[index]
		}
		if err := supervisor.Add(name, worker); err != nil {
			return nil, err
		}
	}
	if err := supervisor.Start(); err != nil {
		return nil, err
	}
	return supervisor, nil
}

func runH3(ctx context.Context, router *gin.Engine) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := tls.GenerateTLSConfig(0, "")
	h3 := http3.Server{
		Addr:       fmt.Sprintf("%s:%d", viper.GetString("listen.host"), viper.GetInt("listen.port")),
		TLSConfig:  cfg,
		QUICConfig: newServerQUICConfig(),
		Handler:    router.Handler(),
	}

	h := http.Server{
		Addr:      fmt.Sprintf("%s:%d", viper.GetString("listen.host"), viper.GetInt("listen.port")),
		TLSConfig: cfg,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := h3.SetQUICHeaders(w.Header())
			if err != nil {
				slog.Error("set HTTP/3 headers failed", "error_code", apperrors.SafeCode(err))
				return
			}
			router.ServeHTTP(w, r)
		}),
	}
	h2Errors := make(chan error, 1)
	go func() {
		err := h.ListenAndServeTLS("", "")
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP/3 companion server failed", "error_code", apperrors.SafeCode(err))
		}
		h2Errors <- err
	}()
	h3Errors := make(chan error, 1)
	go func() { h3Errors <- h3.ListenAndServe() }()

	select {
	case err := <-h3Errors:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if shutdownErr := h.Shutdown(shutdownCtx); shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
			slog.Error("stop HTTP/3 companion server failed", "error_code", apperrors.SafeCode(shutdownErr))
		}
		waitForServerExit(shutdownCtx, "HTTP/3 companion", h2Errors)
		cancel()
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := h3.Shutdown(shutdownCtx); err != nil {
			slog.Error("stop HTTP/3 server failed", "error_code", apperrors.SafeCode(err))
			if closeErr := h3.Close(); closeErr != nil {
				slog.Error("close HTTP/3 server failed", "error_code", apperrors.SafeCode(closeErr))
			}
		}
		if err := h.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("stop HTTP/3 companion server failed", "error_code", apperrors.SafeCode(err))
		}
		waitForServerExit(shutdownCtx, "HTTP/3", h3Errors)
		waitForServerExit(shutdownCtx, "HTTP/3 companion", h2Errors)
		cancel()
		return nil
	case <-h2Errors:
		// The TCP companion failing should not leave the UDP listener orphaned.
		if err := h3.Close(); err != nil {
			slog.Error("close HTTP/3 server after companion failure failed", "error_code", apperrors.SafeCode(err))
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		waitForServerExit(shutdownCtx, "HTTP/3", h3Errors)
		cancel()
		return errors.New("HTTP/3 companion server stopped")
	}
}

// waitForServerExit joins a listener goroutine after its shutdown signal has
// been sent. The bounded context keeps a broken listener from holding process
// shutdown forever, while the normal path makes graceful termination
// deterministic and prevents an orphaned HTTP/2 or HTTP/3 goroutine.
func waitForServerExit(ctx context.Context, name string, done <-chan error) {
	select {
	case <-done:
	case <-ctx.Done():
		slog.Error("wait for server shutdown failed", "server", name, "error_code", "server_shutdown_timeout")
	}
}

func newServerQUICConfig() *quic.Config {
	return &quic.Config{
		KeepAlivePeriod:    time.Second * 10,
		MaxIdleTimeout:     time.Minute * 30,
		MaxIncomingStreams: 100,
		// The application serves non-idempotent requests. Keep 0-RTT disabled
		// so early data cannot replay a login or write operation.
		Allow0RTT:       false,
		EnableDatagrams: true,
	}
}

type H2Handler func(http.ResponseWriter, *http.Request)

func (f H2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}
