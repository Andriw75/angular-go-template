package database

/*
MySQL implementation using go-sql-driver/mysql.

To activate:
  1. Uncomment the code below
  2. Add to go.mod:
       go get github.com/go-sql-driver/mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLPool struct {
	DB *sql.DB
}

func NewMySQLPool(ctx context.Context, dsn string) (*MySQLPool, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &MySQLPool{DB: db}, nil
}

func (p *MySQLPool) Close() error {
	return p.DB.Close()
}

// Example usage in a UserRepository:
//
// type UserRepoMySQL struct {
//     db *sql.DB
// }
//
// func NewUserRepoMySQL(db *sql.DB) *UserRepoMySQL {
//     return &UserRepoMySQL{db: db}
// }
//
// func (r *UserRepoMySQL) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
//     query := `SELECT id, username, email, password, activo FROM usuarios WHERE username = ?`
//     var user domain.User
//     err := r.db.QueryRowContext(ctx, query, username).Scan(
//         &user.ID, &user.Username, &user.Email, &user.Password, &user.Activo,
//     )
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }
*/
