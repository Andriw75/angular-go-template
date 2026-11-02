package handlers

import (
	"back/infrastructure/auth"
	"back/infrastructure/cache"
	"back/infrastructure/storage"
	"back/services"
	"back/store"
)

type Dependencies struct {
	Config           *services.Config
	JWTManager       *auth.JWTManager
	CryptManager     *auth.CryptManager
	JWTStore         *auth.JWTStore
	UserStore        store.UserStore
	PermisoStore     store.PermisoStore
	BusStore         store.BusStore
	MensajeStore     store.MensajeStore
	ActivosStore     *cache.ActivosStore
	CategoriaStore   store.CategoriaStore
	ProductoStore    store.ProductoStore
	ImagenStore      store.ImagenStore
	ArchivoStore     store.ArchivoStore
	Storage          *storage.Storage
	SSEHub           *SSEHub
	InstalacionStore store.InstalacionStore
}
