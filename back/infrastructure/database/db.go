package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"back/services"
)

type DB struct {
	*sql.DB
	dialect Dialect
	dbType  string
}

func Open(cfg *services.Config) (*DB, error) {
	switch cfg.DBType {
	case "postgres":
		db, err := sql.Open("pgx", cfg.DBDSN())
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
		if err := db.Ping(); err != nil {
			db.Close()
			return nil, fmt.Errorf("ping postgres: %w", err)
		}
		return &DB{DB: db, dialect: postgresDialect{}, dbType: "postgres"}, nil

	case "sqlite":
		if dir := filepath.Dir(cfg.DBPath); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create sqlite dir: %w", err)
			}
		}
		db, err := sql.Open("sqlite", cfg.DBPath)
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		db.SetMaxOpenConns(1)
		if err := db.Ping(); err != nil {
			db.Close()
			return nil, fmt.Errorf("ping sqlite: %w", err)
		}
		for _, pragma := range []string{"PRAGMA foreign_keys = ON", "PRAGMA journal_mode = WAL", "PRAGMA busy_timeout = 5000"} {
			if _, err := db.Exec(pragma); err != nil {
				db.Close()
				return nil, fmt.Errorf("sqlite %s: %w", pragma, err)
			}
		}
		return &DB{DB: db, dialect: sqliteDialect{}, dbType: "sqlite"}, nil

	default:
		return nil, fmt.Errorf("DB_TYPE no soportado: %q (usa sqlite o postgres)", cfg.DBType)
	}
}

func (d *DB) Type() string { return d.dbType }

func (d *DB) rebind(query string) string { return d.dialect.Rebind(query) }

func (d *DB) Migrate(ctx context.Context) error {
	for _, stmt := range schema(d.dialect) {
		if _, err := d.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func (d *DB) Seed(ctx context.Context, crypt interface {
	Hash(string) (string, error)
}, cfg *services.Config) error {
	perms := []struct {
		nombre      string
		descripcion string
	}{
		{"buses", "Acceso a buses"},
		{"usuarios", "Gestion de usuarios"},
		{"mensajes_pendientes", "Mensajes pendientes"},
		{"shop", "Gestion de tienda"},
		{"archivos", "Gestion de archivos"},
		{"ventas", "Gestion de ventas"},
		{"reportes", "Generacion de reportes"},
		{"documentos", "Gestion de documentos generados"},
	}

	for _, p := range perms {
		q := d.rebind(`INSERT INTO permissions (nombre, descripcion) VALUES (?, ?) ON CONFLICT (nombre) DO NOTHING`)
		if _, err := d.ExecContext(ctx, q, p.nombre, p.descripcion); err != nil {
			return fmt.Errorf("seed permission %s: %w", p.nombre, err)
		}
	}

	assignAll := func(userID int64) error {
		assign := d.rebind(`INSERT INTO user_permissions (user_id, permission_id)
			SELECT ?, id FROM permissions WHERE nombre = ?
			ON CONFLICT (user_id, permission_id) DO NOTHING`)
		for _, p := range perms {
			if _, err := d.ExecContext(ctx, assign, userID, p.nombre); err != nil {
				return fmt.Errorf("assign permission %s: %w", p.nombre, err)
			}
		}
		return nil
	}

	var count int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		// Reconciliar permisos nuevos para el admin bootstrap, sin tocar otros usuarios.
		var adminID int64
		err := d.QueryRowContext(ctx, d.rebind(`SELECT id FROM users WHERE username = ?`), cfg.BootstrapAdminUsername).Scan(&adminID)
		if err == nil {
			return assignAll(adminID)
		}
		return nil
	}

	// No se crea ningún usuario por defecto. El primer administrador se define
	// explícitamente con BOOTSTRAP_ADMIN_PASSWORD y debe cambiar la contraseña.
	if cfg.BootstrapAdminPassword == "" {
		slog.Warn("base de datos sin usuarios y BOOTSTRAP_ADMIN_PASSWORD vacío: no se creó ningún administrador")
		return nil
	}
	if err := services.ValidatePasswordForUser(cfg.BootstrapAdminUsername, cfg.BootstrapAdminPassword); err != nil {
		return fmt.Errorf("bootstrap admin password: %w", err)
	}

	hash, err := crypt.Hash(cfg.BootstrapAdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	now := formatTime(time.Now())

	q := d.rebind(`INSERT INTO users (username, email, password, activo, must_change_password, creado_en, actualizado_en) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`)
	var userID int64
	if err := d.QueryRowContext(ctx, q,
		cfg.BootstrapAdminUsername, cfg.BootstrapAdminEmail, hash, 1, 1, now, now,
	).Scan(&userID); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	return assignAll(userID)
}

func (d *DB) Close() error { return d.DB.Close() }
