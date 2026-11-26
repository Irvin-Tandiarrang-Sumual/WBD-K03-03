package config_test

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/config"
)

func TestLoadConfig_NoEnvironmentVariablesSet_UsesDefaultValues(t *testing.T) {
	// Arrange
	envKeys := []string{
		"INTERNAL_DB_HOST", "INTERNAL_DB_PORT", "INTERNAL_DB_USER",
		"INTERNAL_DB_PASSWORD", "INTERNAL_DB_NAME",
		"SERVER_HOST", "SERVER_PORT", "FRONTEND_ORIGIN",
		"DATABASE_CREDENTIAL_KEY", "SESSION_COOKIE_SECURE",
		"MIGRATION_WORKER_COUNT", "MIGRATION_QUEUE_CAPACITY",
		"OTEL_SERVICE_NAME", "OTEL_TRACES_EXPORTER", "OTEL_EXPORTER_OTLP_ENDPOINT",
	}
	for _, key := range envKeys {
		t.Setenv(key, "")
	}

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	if err != nil {
		t.Fatalf("unexpected LoadConfig error: %v", err)
	}

	if cfg.Database.Host != "localhost" ||
		cfg.Database.Port != 5432 ||
		cfg.Database.User != "postgresql" ||
		cfg.Database.Password != "" ||
		cfg.Database.Name != "tubes-wbd-3" {
		t.Fatalf("unexpected default database config: %+v", cfg.Database)
	}

	if cfg.Server.Host != "0.0.0.0" ||
		cfg.Server.Port != 8080 ||
		cfg.Server.FrontendOrigin != "http://localhost:3000" {
		t.Fatalf("unexpected default server config: %+v", cfg.Server)
	}

	if cfg.Security.DatabaseCredentialKey != "" {
		t.Fatalf("unexpected default credential key: %q", cfg.Security.DatabaseCredentialKey)
	}

	if !cfg.Session.CookieSecure {
		t.Fatal("expected Session.CookieSecure default = true, got false")
	}

	if cfg.Worker.Count != 5 {
		t.Fatalf("expected Worker.Count default = 5, got %d", cfg.Worker.Count)
	}

	if cfg.Worker.QueueCapacity != 100 {
		t.Fatalf("expected Worker.QueueCapacity default = 100, got %d", cfg.Worker.QueueCapacity)
	}

	if cfg.Telemetry.ServiceName != "migration-service" {
		t.Fatalf("expected Telemetry.ServiceName default = 'migration-service', got %s", cfg.Telemetry.ServiceName)
	}

	if cfg.Telemetry.Exporter != "console" {
		t.Fatalf("expected Telemetry.Exporter default = 'console', got %s", cfg.Telemetry.Exporter)
	}

	if cfg.Telemetry.Endpoint != "localhost:4317" {
		t.Fatalf("expected Telemetry.Endpoint default = 'localhost:4317', got %s", cfg.Telemetry.Endpoint)
	}
}

func TestLoadConfig_ValidEnvironmentVariablesSet_PopulatesConfigValues(t *testing.T) {
	// Arrange
	t.Setenv("INTERNAL_DB_HOST", "db.internal")
	t.Setenv("INTERNAL_DB_PORT", "6432")
	t.Setenv("INTERNAL_DB_USER", "app_user")
	t.Setenv("INTERNAL_DB_PASSWORD", "supersecret")
	t.Setenv("INTERNAL_DB_NAME", "production_db")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("FRONTEND_ORIGIN", "https://app.example.com")
	t.Setenv("DATABASE_CREDENTIAL_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	t.Setenv("MIGRATION_WORKER_COUNT", "10")
	t.Setenv("MIGRATION_QUEUE_CAPACITY", "250")
	t.Setenv("OTEL_SERVICE_NAME", "custom-migration-service")
	t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector.internal:4317")

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	if err != nil {
		t.Fatalf("unexpected LoadConfig error: %v", err)
	}

	if cfg.Database.Host != "db.internal" ||
		cfg.Database.Port != 6432 ||
		cfg.Database.User != "app_user" ||
		cfg.Database.Password != "supersecret" ||
		cfg.Database.Name != "production_db" {
		t.Fatalf("unexpected database config: %+v", cfg.Database)
	}

	if cfg.Server.Host != "127.0.0.1" ||
		cfg.Server.Port != 9090 ||
		cfg.Server.FrontendOrigin != "https://app.example.com" {
		t.Fatalf("unexpected server config: %+v", cfg.Server)
	}

	if cfg.Security.DatabaseCredentialKey != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected credential key: %q", cfg.Security.DatabaseCredentialKey)
	}

	if cfg.Session.CookieSecure != false {
		t.Fatal("expected Session.CookieSecure = false, got true")
	}

	if cfg.Worker.Count != 10 {
		t.Fatalf("expected Worker.Count = 10, got %d", cfg.Worker.Count)
	}

	if cfg.Worker.QueueCapacity != 250 {
		t.Fatalf("expected Worker.QueueCapacity = 250, got %d", cfg.Worker.QueueCapacity)
	}

	if cfg.Telemetry.ServiceName != "custom-migration-service" {
		t.Fatalf("expected Telemetry.ServiceName = 'custom-migration-service', got %s", cfg.Telemetry.ServiceName)
	}

	if cfg.Telemetry.Exporter != "otlp" {
		t.Fatalf("expected Telemetry.Exporter = 'otlp', got %s", cfg.Telemetry.Exporter)
	}

	if cfg.Telemetry.Endpoint != "collector.internal:4317" {
		t.Fatalf("expected Telemetry.Endpoint = 'collector.internal:4317', got %s", cfg.Telemetry.Endpoint)
	}
}

func TestLoadConfig_InvalidEnvironmentVariablesSet_FallsBackToDefaults(t *testing.T) {
	// Arrange
	t.Setenv("INTERNAL_DB_PORT", "not-a-number")
	t.Setenv("SERVER_PORT", "invalid-port")
	t.Setenv("SESSION_COOKIE_SECURE", "not-a-boolean")
	t.Setenv("MIGRATION_WORKER_COUNT", "not-an-int")
	t.Setenv("MIGRATION_QUEUE_CAPACITY", "also-not-an-int")

	// Act
	cfg, err := config.LoadConfig()

	// Assert
	if err != nil {
		t.Fatalf("unexpected LoadConfig error: %v", err)
	}

	if cfg.Database.Port != 5432 {
		t.Fatalf("expected fallback Database.Port = 5432, got %d", cfg.Database.Port)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected fallback Server.Port = 8080, got %d", cfg.Server.Port)
	}

	if !cfg.Session.CookieSecure {
		t.Fatal("expected fallback Session.CookieSecure = true, got false")
	}

	if cfg.Worker.Count != 5 {
		t.Fatalf("expected fallback Worker.Count = 5, got %d", cfg.Worker.Count)
	}

	if cfg.Worker.QueueCapacity != 100 {
		t.Fatalf("expected fallback Worker.QueueCapacity = 100, got %d", cfg.Worker.QueueCapacity)
	}
}
