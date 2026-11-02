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
	"back/infrastructure/cache"
	"back/infrastructure/database"
	"back/infrastructure/storage"
	"back/services"
	"back/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := services.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	jwtStore := auth.NewJWTStore()
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpirationMin, cfg.JWTRenewMin)
	cryptManager := auth.NewCryptManager()

	if cfg.JWTStoreEnabled {
		jwtManager.Store = jwtStore
	}

	db, err := database.Open(cfg)
	if err != nil {
		slog.Error("failed to open database", "error", err, "type", cfg.DBType)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		slog.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}
	if err := db.Seed(ctx, cryptManager); err != nil {
		slog.Error("failed to seed database", "error", err)
		os.Exit(1)
	}
	slog.Info("database ready", "type", db.Type())

	activos := cache.NewActivosStore()

	deps := &handlers.Dependencies{
		Config:           cfg,
		JWTManager:       jwtManager,
		CryptManager:     cryptManager,
		JWTStore:         jwtStore,
		UserStore:        database.NewUserStore(db, cryptManager),
		PermisoStore:     database.NewPermisoStore(db),
		BusStore:         database.NewBusStore(db),
		MensajeStore:     database.NewMensajeStore(db),
		ActivosStore:     activos,
		CategoriaStore:   database.NewCategoriaStore(db),
		ProductoStore:    database.NewProductoStore(db),
		ImagenStore:      database.NewImagenStore(db),
		ArchivoStore:     database.NewArchivoStore(db),
		InstalacionStore: database.NewInstalacionStore(db),
		SSEHub:           handlers.NewSSEHub(),
		Storage:          storage.NewStorage(cfg.ImagesDir()),
	}
	seedActivos(deps.MensajeStore, activos)

	if err := deps.Storage.EnsureDirs(); err != nil {
		slog.Error("failed to create images dir", "error", err, "dir", cfg.ImagesDir())
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.FilesDir(), 0o755); err != nil {
		slog.Error("failed to create files dir", "error", err, "dir", cfg.FilesDir())
		os.Exit(1)
	}
	slog.Info("files dir ready", "dir", cfg.FilesDir(), "x_accel_redirect", cfg.XAccelRedirectPrefix != "")

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}

func seedActivos(ms store.MensajeStore, activos *cache.ActivosStore) {
	for _, m := range ms.FindAll() {
		if m.Estado != "finalizado" {
			activos.Set(m)
		}
	}
}
