package repository

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	usersTable      = "users"
	todoListsTable  = "todo_lists"
	usersListsTable = "users_lists"
	todoItemsTable  = "todo_items"
	listsItemsTable = "lists_items"
)

type Config struct{ Host, Port, Username, Password, DBName, SSLMode string }

func NewPostgresDb(cfg Config) (*sqlx.DB, error) {
	dsn := url.URL{Scheme: "postgres", Host: net.JoinHostPort(cfg.Host, cfg.Port), User: url.UserPassword(cfg.Username, cfg.Password), Path: "/" + cfg.DBName}
	query := url.Values{"sslmode": {cfg.SSLMode}, "connect_timeout": {"5"}, "options": {"-c statement_timeout=10000"}}
	dsn.RawQuery = query.Encode()
	db, err := sqlx.Open("postgres", dsn.String())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
