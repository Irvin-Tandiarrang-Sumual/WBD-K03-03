package postgresql

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ConnectionTester struct{}

var _ ports.DatabaseConnectionTester = (*ConnectionTester)(nil)

func NewConnectionTester() *ConnectionTester {
	return &ConnectionTester{}
}

func (t *ConnectionTester) Test(ctx context.Context, database domain.Database) error {
	dsn := buildDSN(database)

	// First attempt: try connecting directly to the target database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err == nil {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			defer sqlDB.Close()
			if pingErr := sqlDB.PingContext(ctx); pingErr == nil {
				return nil
			}
		}
	}

	// If direct connect failed, check if we can connect to the postgres server and create the database dynamically
	if err := createDatabaseIfNotExists(ctx, database); err != nil {
		return err
	}

	// Verify connection to the new/verified database
	verifyDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open PostgreSQL connection: %w", err)
	}

	sqlDB, err := verifyDB.DB()
	if err != nil {
		return fmt.Errorf("get PostgreSQL SQL connection: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL database: %w", err)
	}

	return nil
}

func createDatabaseIfNotExists(ctx context.Context, database domain.Database) error {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, database.Name)
	if !matched {
		return fmt.Errorf("invalid database name %q: only alphanumeric and underscore characters are allowed", database.Name)
	}

	// Connect to default administrative database ("postgres" or "template1")
	maintDb := database
	maintDb.Name = "postgres"
	maintDSN := buildDSN(maintDb)

	maintConn, err := gorm.Open(postgres.Open(maintDSN), &gorm.Config{})
	if err != nil {
		maintDb.Name = "template1"
		maintDSN = buildDSN(maintDb)
		maintConn, err = gorm.Open(postgres.Open(maintDSN), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("connect to PostgreSQL server: %w", err)
		}
	}

	sqlDB, err := maintConn.DB()
	if err != nil {
		return fmt.Errorf("get PostgreSQL connection: %w", err)
	}
	defer sqlDB.Close()

	var count int
	row := sqlDB.QueryRowContext(ctx, "SELECT count(1) FROM pg_database WHERE datname = $1", database.Name)
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("check PostgreSQL database: %w", err)
	}

	if count == 0 {
		query := fmt.Sprintf(`CREATE DATABASE "%s"`, database.Name)
		if _, err := sqlDB.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("create PostgreSQL database %q: %w", database.Name, err)
		}
	}

	return nil
}

func buildDSN(database domain.Database) string {
	connectionURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(database.User, database.Password),
		Host:   net.JoinHostPort(database.Host, strconv.Itoa(int(database.Port))),
		Path:   database.Name,
	}
	query := connectionURL.Query()
	query.Set("sslmode", "disable")
	connectionURL.RawQuery = query.Encode()
	return connectionURL.String()
}
