package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ginprojectapi/internal/config"
	"ginprojectapi/internal/database"
	httpapi "ginprojectapi/internal/http"
	"ginprojectapi/internal/service"
	"ginprojectapi/internal/store"
	"ginprojectapi/internal/store/memory"
	"ginprojectapi/internal/store/sqlserver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	repositories, cleanup, err := buildRepositories(cfg, logger)
	if err != nil {
		logger.Error("repository initialization failed", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	jwtManager := service.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer, cfg.Auth.AccessTokenTTL)
	services := service.Services{
		Auth:    service.NewAuthService(repositories.Users, jwtManager),
		Product: service.NewProductService(repositories.Products),
		Cart:    service.NewCartService(repositories.Products, repositories.Carts),
		JWT:     jwtManager,
	}

	router := httpapi.NewRouter(cfg, services, logger)
	server := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api listening", "addr", server.Addr, "environment", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("api stopped")
}

func buildRepositories(cfg config.Config, logger *slog.Logger) (store.Repositories, func(), error) {
	if cfg.Database.DSN == "" {
		logger.Warn("DATABASE_DSN is empty; using in-memory repositories for local development")
		repositories := memory.NewRepositories()
		return repositories, func() {}, nil
	}

	db, err := database.OpenSQLServer(cfg.Database)
	if err != nil {
		return store.Repositories{}, func() {}, err
	}

	repositories := sqlserver.NewRepositories(db)
	return repositories, func() {
		if err := db.Close(); err != nil {
			logger.Warn("database close failed", "error", err)
		}
	}, nil
}
