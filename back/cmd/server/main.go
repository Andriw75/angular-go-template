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
	"back/domain"
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
	loginLimiter := auth.NewLoginLimiter(
		cfg.LoginMaxAttempts,
		time.Duration(cfg.LoginWindowMinutes)*time.Minute,
		time.Duration(cfg.LoginBlockMinutes)*time.Minute,
	)

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
	if err := db.Seed(ctx, cryptManager, cfg); err != nil {
		slog.Error("failed to seed database", "error", err)
		os.Exit(1)
	}
	slog.Info("database ready", "type", db.Type())

	activos := cache.NewActivosStore()
	sseHub := handlers.NewSSEHub()

	deps := &handlers.Dependencies{
		Config:           cfg,
		JWTManager:       jwtManager,
		CryptManager:     cryptManager,
		JWTStore:         jwtStore,
		LoginLimiter:     loginLimiter,
		UserStore:        database.NewUserStore(db, cryptManager),
		PermisoStore:     database.NewPermisoStore(db),
		BusStore:         database.NewBusStore(db),
		MensajeStore:     database.NewMensajeStore(db),
		ActivosStore:     activos,
		CategoriaStore:   database.NewCategoriaStore(db),
		ProductoStore:    database.NewProductoStore(db),
		ImagenStore:      database.NewImagenStore(db),
		ArchivoStore:     database.NewArchivoStore(db),
		VentaStore:       database.NewVentaStore(db),
		DocumentoStore:   database.NewDocumentoStore(db),
		InstalacionStore: database.NewInstalacionStore(db),
		SSEHub:           sseHub,
		Storage:          storage.NewStorage(cfg.PrivateImagesDir()),
		SlowQueryStore:   database.NewLabStore(db, handlers.NewLabNotifier(sseHub), 10, 5, 600, 2_000_000),
	}
	seedActivos(deps.MensajeStore, activos)

	dirs := []string{
		cfg.PublicImagesDir(),
		cfg.PrivateImagesDir(),
		cfg.PublicFilesDir(),
		cfg.PrivateFilesDir(),
		cfg.DocumentosDir(),
	}
	if err := deps.Storage.EnsureDirsFor(dirs...); err != nil {
		slog.Error("failed to create storage dirs", "error", err)
		os.Exit(1)
	}
	syncPlacement(deps)
	slog.Info("storage dirs ready", "public_images", cfg.PublicImagesDir(), "private_images", cfg.PrivateImagesDir(), "public_files", cfg.PublicFilesDir(), "private_files", cfg.PrivateFilesDir(), "x_accel_redirect", cfg.XAccelRedirectPrefix != "")

	router := api.NewRouter(deps)

	srv := &http.Server{
		Addr:              cfg.ServerPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
		// WriteTimeout se deja en 0 porque /sse/events es un stream SSE
		// (un stream de larga duración). Los handlers que lo necesiten pueden
		// fijar deadlines por respuesta con http.NewResponseController.
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

	deps.SlowQueryStore.CancelAll(shutdownCtx)

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

// syncPlacement mueve archivos existentes a la carpeta que les corresponde
// (pública para nginx, privada para la API) al arrancar.
func syncPlacement(deps *handlers.Dependencies) {
	for _, img := range deps.ImagenStore.ListAll() {
		public, err := handlers.ImageIsPublic(deps, img.Tipo, img.EntidadID)
		if err != nil {
			continue
		}
		src, dst := deps.Config.PrivateImagesDir(), deps.Config.PublicImagesDir()
		if !public {
			src, dst = dst, src
		}
		if err := deps.Storage.MoveFileIfMissing(src, dst, img.FileName); err != nil {
			slog.Warn("no se pudo ubicar imagen", "file", img.FileName, "error", err)
		}
	}

	files, _, err := deps.ArchivoStore.List(0, 1000000, "")
	if err != nil {
		return
	}
	for _, a := range files {
		public := a.Visibilidad == domain.VisibilidadPublico
		src, dst := deps.Config.PrivateFilesDir(), deps.Config.PublicFilesDir()
		if !public {
			src, dst = dst, src
		}
		if err := deps.Storage.MoveFileIfMissing(src, dst, a.FileName); err != nil {
			slog.Warn("no se pudo ubicar archivo", "file", a.FileName, "error", err)
		}
	}
}
