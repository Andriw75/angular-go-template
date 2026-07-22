package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"back/api"
	"back/api/handlers"
	"back/infrastructure/auth"
	"back/mock"
	"back/services"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := services.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpirationMin)
	cryptManager := auth.NewCryptManager()

	deps := &handlers.Dependencies{
		Config:       cfg,
		JWTManager:   jwtManager,
		CryptManager: cryptManager,
	}

	if cfg.UseMock {
		slog.Info("using mock data store")
		deps.UserStore = mock.NewUserStore(cryptManager)
		deps.PermisoStore = mock.NewPermisoStore()
	} else {
		slog.Info("connecting to database", "type", cfg.DBType)
		// TODO: Initialize real database repositories when USE_MOCK=false
		// See infrastructure/database/ for implementations
		deps.UserStore = mock.NewUserStore(cryptManager)
		deps.PermisoStore = mock.NewPermisoStore()
	}

	router := api.NewRouter(deps)

	srv := &http.Server{
		Addr:    cfg.ServerPort,
		Handler: router,
	}

	go func() {
		slog.Info("server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}
