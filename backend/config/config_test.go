package config

import "testing"

func TestLoadConfigUsesDefaults(t *testing.T) {
	for _, key := range []string{
		"INTERNAL_DB_HOST", "INTERNAL_DB_PORT", "INTERNAL_DB_USER", "INTERNAL_DB_PASSWORD", "INTERNAL_DB_NAME",
		"SERVER_HOST", "SERVER_PORT", "FRONTEND_ORIGIN", "DATABASE_CREDENTIAL_KEY", "SESSION_COOKIE_SECURE", "SEED_TEST_DATA",
	} {
		t.Setenv(key, "")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Database.Host != "localhost" || cfg.Database.Port != 5432 || cfg.Database.User != "postgresql" || cfg.Database.Password != "" || cfg.Database.Name != "tubes-wbd-3" {
		t.Fatalf("unexpected default database config: %+v", cfg.Database)
	}
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != 8080 || cfg.Server.FrontendOrigin != "http://localhost:3000" {
		t.Fatalf("unexpected default server config: %+v", cfg.Server)
	}
	if cfg.Security.DatabaseCredentialKey != "" {
		t.Fatalf("unexpected default credential key: %q", cfg.Security.DatabaseCredentialKey)
	}
	if cfg.Session.CookieSecure {
		t.Fatal("CookieSecure default = true, want false")
	}
	if cfg.SeedTestData {
		t.Fatal("SeedTestData default = true, want false")
	}
}

func TestLoadConfigReadsEnvironment(t *testing.T) {
	t.Setenv("INTERNAL_DB_HOST", "db.internal")
	t.Setenv("INTERNAL_DB_PORT", "6432")
	t.Setenv("INTERNAL_DB_USER", "app")
	t.Setenv("INTERNAL_DB_PASSWORD", "secret")
	t.Setenv("INTERNAL_DB_NAME", "platform")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("FRONTEND_ORIGIN", "https://frontend.example.com")
	t.Setenv("DATABASE_CREDENTIAL_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	t.Setenv("SEED_TEST_DATA", "true")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Database.Host != "db.internal" || cfg.Database.Port != 6432 || cfg.Database.User != "app" || cfg.Database.Password != "secret" || cfg.Database.Name != "platform" {
		t.Fatalf("unexpected database config: %+v", cfg.Database)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 9090 || cfg.Server.FrontendOrigin != "https://frontend.example.com" {
		t.Fatalf("unexpected server config: %+v", cfg.Server)
	}
	if cfg.Security.DatabaseCredentialKey != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected credential key: %q", cfg.Security.DatabaseCredentialKey)
	}
	if !cfg.Session.CookieSecure {
		t.Fatal("CookieSecure = false, want true")
	}
	if !cfg.SeedTestData {
		t.Fatal("SeedTestData = false, want true")
	}
}

func TestInvalidEnvironmentValuesFallBackToDefaults(t *testing.T) {
	t.Setenv("INTERNAL_DB_PORT", "not-a-number")
	t.Setenv("SERVER_PORT", "invalid")
	t.Setenv("SESSION_COOKIE_SECURE", "not-a-bool")
	t.Setenv("SEED_TEST_DATA", "not-a-bool")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Database.Port != 5432 {
		t.Fatalf("Database.Port = %d, want 5432", cfg.Database.Port)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Session.CookieSecure {
		t.Fatal("CookieSecure = true, want fallback false")
	}
	if cfg.SeedTestData {
		t.Fatal("SeedTestData = true, want fallback false")
	}
}
