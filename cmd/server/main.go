package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"avagostar-form-server/internal/config"
	"avagostar-form-server/internal/db"
	transport "avagostar-form-server/internal/http"
	"avagostar-form-server/internal/http/middleware"
	"avagostar-form-server/internal/repo"
	"avagostar-form-server/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.Env)

	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	dbConn, err := db.Connect(ctx, cfg.DBURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	if err := db.EnsureSeedUsers(ctx, dbConn.Pool, cfg.RequestTimeout); err != nil {
		logger.Error("failed to seed users", "error", err)
		os.Exit(1)
	}

	userRepo := repo.NewUserRepo(dbConn.Pool, cfg.RequestTimeout)
	txRepo := repo.NewTransactionRepo(dbConn.Pool, cfg.RequestTimeout)

	authService := services.NewAuthService(userRepo, cfg)
	txService := services.NewTransactionService(txRepo)

	router := transport.NewRouter(transport.Dependencies{
		Config:      cfg,
		UserRepo:    userRepo,
		AuthService: authService,
		TxService:   txService,
		Logger:      logger,
		RateLimiter: middleware.NewRateLimiter(cfg.RateLimitPerMinute),
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadTimeout:       cfg.RequestTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      cfg.RequestTimeout,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "addr", cfg.HTTPAddr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		logger.Error("http server stopped unexpectedly", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("http server stopped")
}

func newLogger(env string) *slog.Logger {
	level := slog.LevelInfo
	if env != "prod" {
		level = slog.LevelDebug
	}

	var handler slog.Handler
	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	return slog.New(handler)
}
