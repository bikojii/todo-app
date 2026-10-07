package repository

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestPostgresConnection(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL with a user")
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	mode := u.Query().Get("sslmode")
	if mode == "" {
		mode = "disable"
	}
	password, _ := u.User.Password()
	db, err := NewPostgresDb(Config{
		Host: u.Hostname(), Port: port, Username: u.User.Username(), Password: password,
		DBName: strings.TrimPrefix(u.Path, "/"), SSLMode: mode,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var timeout string
	if err := db.Get(&timeout, "SHOW statement_timeout"); err != nil {
		t.Fatal(err)
	}
	if timeout != "10s" {
		t.Fatalf("statement_timeout = %q", timeout)
	}
}
