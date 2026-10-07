package main

import "testing"

func TestConfigEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_PASSWORD", "test-password")
	if _, err := loadConfig(); err == nil {
		t.Fatal("accepted missing secret")
	}
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	t.Setenv("PORT", "9000")
	t.Setenv("DB_PORT", "5433")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetString("port") != "9000" || cfg.GetString("db.port") != "5433" || cfg.GetString("db.password") != "test-password" {
		t.Fatal("environment ignored")
	}
	t.Setenv("PORT", "-1")
	if _, err := loadConfig(); err == nil {
		t.Fatal("accepted invalid port")
	}
}
