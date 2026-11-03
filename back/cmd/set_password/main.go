package main

import (
	"flag"
	"fmt"
	"os"

	"back/infrastructure/auth"
	"back/infrastructure/database"
	"back/services"
)

// set_password rota la contraseña de un usuario existente. Útil para limpiar
// credenciales antiguas (por ejemplo, un admin creado con el seed anterior).
//
// Uso (desde back/):
//   NEW_PASSWORD='...' go run ./cmd/set_password -user USUARIO_ADMIN
// o:
//   go run ./cmd/set_password -user USUARIO_ADMIN -password '...'
func main() {
	username := flag.String("user", "", "usuario a actualizar")
	password := flag.String("password", "", "nueva contraseña (mejor usar NEW_PASSWORD)")
	flag.Parse()

	if *username == "" {
		fmt.Fprintln(os.Stderr, "falta -user")
		os.Exit(2)
	}

	newPassword := *password
	if newPassword == "" {
		newPassword = os.Getenv("NEW_PASSWORD")
	}
	if newPassword == "" {
		fmt.Fprintln(os.Stderr, "define NEW_PASSWORD o usa -password")
		os.Exit(2)
	}
	if err := services.ValidatePasswordForUser(*username, newPassword); err != nil {
		fmt.Fprintln(os.Stderr, "contraseña inválida:", err)
		os.Exit(1)
	}

	cfg, err := services.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer db.Close()

	crypt := auth.NewCryptManager()
	users := database.NewUserStore(db, crypt)

	user, err := users.FindByUsername(*username)
	if err != nil {
		fmt.Fprintln(os.Stderr, "usuario no encontrado:", err)
		os.Exit(1)
	}

	hash, err := crypt.Hash(newPassword)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash:", err)
		os.Exit(1)
	}
	if err := users.UpdatePassword(user.ID, hash, false); err != nil {
		fmt.Fprintln(os.Stderr, "update:", err)
		os.Exit(1)
	}

	fmt.Printf("contraseña actualizada para %q (id=%d)\n", user.Username, user.ID)
}
