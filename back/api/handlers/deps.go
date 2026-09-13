package handlers

import (
	"back/infrastructure/auth"
	"back/mock"
	"back/services"
)

type Dependencies struct {
	Config       *services.Config
	JWTManager   *auth.JWTManager
	CryptManager *auth.CryptManager
	JWTStore     *auth.JWTStore
	UserStore    *mock.UserStore
	PermisoStore *mock.PermisoStore
}
