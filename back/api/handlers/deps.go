package handlers

import (
	"back/infrastructure/auth"
	"back/infrastructure/storage"
	"back/mock"
	"back/services"
)

type Dependencies struct {
	Config           *services.Config
	JWTManager       *auth.JWTManager
	CryptManager     *auth.CryptManager
	JWTStore         *auth.JWTStore
	UserStore        *mock.UserStore
	PermisoStore     *mock.PermisoStore
	BusStore         *mock.BusStore
	MensajeStore     *mock.MensajeStore
	ActivosStore     *mock.ActivosStore
	CategoriaStore   *mock.CategoriaStore
	ProductoStore    *mock.ProductoStore
	ImagenStore      *mock.ImagenStore
	Storage          *storage.Storage
	SSEHub           *SSEHub
	InstalacionStore *mock.InstalacionStore
}
