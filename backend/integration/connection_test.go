//go:build integration

package integration_test

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	postgresqldriver "github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/driver/postgresql"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func TestIntegrationPostgreSQLConnectionTester(t *testing.T) {
	dsn := os.Getenv("PLATFORM_DATABASE_URL")

	if dsn == "" {
		t.Skip(
			"PLATFORM_DATABASE_URL is not set; skipping PostgreSQL integration test",
		)
	}

	parsed, err := url.Parse(dsn)

	if err != nil {
		t.Fatalf(
			"parse PLATFORM_DATABASE_URL: %v",
			err,
		)
	}

	port := uint(5432)

	if parsed.Port() != "" {
		value, err := strconv.ParseUint(
			parsed.Port(),
			10,
			16,
		)

		if err != nil {
			t.Fatalf(
				"invalid port in PLATFORM_DATABASE_URL: %v",
				err,
			)
		}

		port = uint(value)
	}

	password, _ := parsed.User.Password()

	database := domain.Database{
		Host:     parsed.Hostname(),
		Port:     port,
		User:     parsed.User.Username(),
		Password: password,
		Type:     "postgresql",
		Name:     strings.TrimPrefix(
			parsed.Path,
			"/",
		),
	}

	tester := postgresqldriver.NewConnectionTester()

	if err := tester.Test(
		context.Background(),
		database,
	); err != nil {
		t.Fatalf(
			"ConnectionTester.Test() error = %v",
			err,
		)
	}
}
