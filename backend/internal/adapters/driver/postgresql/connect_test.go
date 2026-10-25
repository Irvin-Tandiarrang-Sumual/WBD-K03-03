package postgresql

import (
	"net/url"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestBuildDSN(t *testing.T) {
	database := domain.Database{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "secret@123",
		Name:     "target_db",
	}

	dsn := buildDSN(database)

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	if parsed.Scheme != "postgres" {
		t.Fatalf("scheme = %q, want %q", parsed.Scheme, "postgres")
	}

	if parsed.Host != "localhost:5432" {
		t.Fatalf("host = %q, want %q", parsed.Host, "localhost:5432")
	}

	if parsed.Path != "/target_db" {
		t.Fatalf("path = %q, want %q", parsed.Path, "/target_db")
	}

	username := parsed.User.Username()

	if username != "postgres" {
		t.Fatalf("username = %q, want %q", username, "postgres")
	}

	password, ok := parsed.User.Password()

	if !ok {
		t.Fatal("password is missing")
	}

	if password != "secret@123" {
		t.Fatalf("password = %q, want %q", password, "secret@123")
	}

	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("sslmode = %q, want %q", parsed.Query().Get("sslmode"), "disable")
	}
}