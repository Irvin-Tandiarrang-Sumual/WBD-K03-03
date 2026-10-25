package mysql

import (
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestBuildDSNIncludesConnectionFields(t *testing.T) {
	database := domain.Database{
		Host:     "127.0.0.1",
		Port:     3307,
		User:     "app user",
		Password: "p@ss:word",
		Name:     "sample_db",
	}

	dsn := buildDSN(database)
	for _, part := range []string{"app user:", "@tcp(127.0.0.1:3307)/sample_db", "parseTime=true"} {
		if !strings.Contains(dsn, part) {
			t.Fatalf("buildDSN() = %q, want it to contain %q", dsn, part)
		}
	}
}
