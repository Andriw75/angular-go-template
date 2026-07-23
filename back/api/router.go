package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"back/api/handlers"
)

func NewRouter(deps *handlers.Dependencies) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware(deps.Config.CORSOrigin))

	authHandler := handlers.NewAuthHandler(deps)
	userHandler := handlers.NewUserHandler(deps)
	permisoHandler := handlers.NewPermisoHandler(deps)
	busHandler := handlers.NewBusHandler(deps)
	mensajeHandler := handlers.NewMensajeHandler(deps)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/token", authHandler.Token)
		r.Group(func(r chi.Router) {
			r.Use(authHandler.AuthMiddleware())
			r.Post("/logout", authHandler.Logout)
			r.Get("/me", authHandler.Me)
		})
	})

	r.Route("/users", func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware("usuarios"))
		r.Get("/", userHandler.List)
		r.Post("/", userHandler.Create)
		r.Get("/{id}", userHandler.GetByID)
		r.Put("/{id}", userHandler.Update)
		r.Delete("/{id}", userHandler.Delete)
	})

	r.Route("/permisos", func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware("usuarios"))
		r.Get("/", permisoHandler.List)
	})

	r.Route("/buses", func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware("buses"))
		r.Get("/count", busHandler.Count)
		r.Get("/", busHandler.List)
		r.Post("/", busHandler.Create)
		r.Get("/{id}", busHandler.GetByID)
		r.Put("/{id}", busHandler.Update)
		r.Delete("/{id}", busHandler.Delete)
	})

	r.Route("/mensajes_pendientes", func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware("mensajes_pendientes"))
		r.Get("/events", mensajeHandler.Events)
		r.Get("/", mensajeHandler.List)
		r.Post("/", mensajeHandler.Create)
		r.Get("/pasadas", mensajeHandler.ListPasadas)
		r.Get("/pasadas/count", mensajeHandler.CountPasadas)
		r.Get("/{id}", mensajeHandler.GetByID)
		r.Put("/{id}", mensajeHandler.Update)
		r.Delete("/{id}", mensajeHandler.Delete)
	})

	return r
}

func corsMiddleware(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
