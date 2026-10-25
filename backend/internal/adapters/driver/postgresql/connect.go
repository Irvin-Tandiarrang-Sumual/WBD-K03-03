package postgresql

import (
	"context"
	"fmt"
	"net"
	"net/url"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open PostgreSQL connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get PostgreSQL SQL connection: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL database: %w", err)
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
