package mysql

import (
	"context"
	"fmt"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ConnectionTester struct{}

var _ ports.DatabaseConnectionTester = (*ConnectionTester)(nil)

func NewConnectionTester() *ConnectionTester {
	return &ConnectionTester{}
}

func (t *ConnectionTester) Test(ctx context.Context, database domain.Database) error {
	db, err := gorm.Open(mysql.Open(buildDSN(database)), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open MySQL connection: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get MySQL SQL connection: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping MySQL database: %w", err)
	}
	return nil
}

func buildDSN(database domain.Database) string {
	config := mysqlDriver.Config{
		User:      database.User,
		Passwd:    database.Password,
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%d", database.Host, database.Port),
		DBName:    database.Name,
		ParseTime: true,
	}
	return config.FormatDSN()
}
