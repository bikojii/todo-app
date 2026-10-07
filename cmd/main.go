package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	todo "github.com/bikojii/todo-app"
	"github.com/bikojii/todo-app/pkg/handler"
	"github.com/bikojii/todo-app/pkg/repository"
	"github.com/bikojii/todo-app/pkg/service"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	logrus.SetFormatter(new(logrus.JSONFormatter))
	if err := run(); err != nil {
		logrus.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	db, err := repository.NewPostgresDb(repository.Config{
		Host: cfg.GetString("db.host"), Port: cfg.GetString("db.port"), Username: cfg.GetString("db.username"),
		DBName: cfg.GetString("db.dbname"), Password: cfg.GetString("db.password"), SSLMode: cfg.GetString("db.sslmode"),
	})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()
	services, err := service.NewService(repository.NewRepository(db), cfg.GetString("jwt_secret"))
	if err != nil {
		return err
	}
	srv := todo.NewServer(cfg.GetString("port"), handler.NewHandler(services).InitRoutes())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() { serverErr <- srv.Run() }()
	logrus.WithField("port", cfg.GetString("port")).Info("TodoApp started")
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		listenErr := <-serverErr
		if err != nil {
			return err
		}
		if !errors.Is(listenErr, http.ErrServerClosed) {
			return listenErr
		}
		return nil
	}
}

func loadConfig() (*viper.Viper, error) {
	cfg := viper.New()
	cfg.AddConfigPath("configs")
	cfg.SetConfigName("config")
	if err := cfg.ReadInConfig(); err != nil {
		var missing viper.ConfigFileNotFoundError
		if !errors.As(err, &missing) {
			return nil, err
		}
	}
	defaults := map[string]interface{}{"port": "8000", "db.host": "localhost", "db.port": "5432", "db.username": "postgres", "db.dbname": "todo", "db.sslmode": "disable"}
	for key, value := range defaults {
		cfg.SetDefault(key, value)
	}
	bindings := map[string]string{"port": "PORT", "db.host": "DB_HOST", "db.port": "DB_PORT", "db.username": "DB_USER", "db.dbname": "DB_NAME", "db.password": "DB_PASSWORD", "db.sslmode": "DB_SSLMODE", "jwt_secret": "JWT_SECRET"}
	for key, env := range bindings {
		if err := cfg.BindEnv(key, env); err != nil {
			return nil, err
		}
	}
	for _, key := range []string{"port", "db.port"} {
		port, err := strconv.Atoi(cfg.GetString(key))
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid %s", key)
		}
	}
	if len(cfg.GetString("jwt_secret")) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	for _, key := range []string{"db.host", "db.username", "db.dbname", "db.password"} {
		if cfg.GetString(key) == "" {
			return nil, fmt.Errorf("%s is required", key)
		}
	}
	return cfg, nil
}
