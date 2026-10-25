//go:build integration

package postgresql

import (
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
)

func TestConnectToSystemDatabaseWithDedicatedIntegrationDatabase(t *testing.T) {
	rawURL := os.Getenv("PLATFORM_DATABASE_URL")
	if rawURL == "" {
		t.Skip("PLATFORM_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse PLATFORM_DATABASE_URL: %v", err)
	}
	password, _ := parsed.User.Password()
	port := 5432
	if parsed.Port() != "" {
		port, err = strconv.Atoi(parsed.Port())
		if err != nil {
			t.Fatalf("invalid PostgreSQL port in PLATFORM_DATABASE_URL: %v", err)
		}
	}

	db, err := ConnectToSystemDatabase(config.DBConfig{
		Host:     parsed.Hostname(),
		Port:     port,
		User:     parsed.User.Username(),
		Password: password,
		Name:     parsed.Path[1:],
	})
	if err != nil {
		t.Fatalf("ConnectToSystemDatabase() error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}
