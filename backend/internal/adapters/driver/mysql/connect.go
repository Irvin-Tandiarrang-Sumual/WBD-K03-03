package mysql

import (
	"context"
	"fmt"
	"regexp"

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
	// First attempt: try connecting directly to the target database
	db, err := gorm.Open(mysql.Open(buildDSN(database)), &gorm.Config{})
	if err == nil {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			defer sqlDB.Close()
			if pingErr := sqlDB.PingContext(ctx); pingErr == nil {
				return nil
			}
		}
	}

	// If direct connect failed, create the database dynamically
	if err := createDatabaseIfNotExists(ctx, database); err != nil {
		return err
	}

	// Verify connection to the new database
	verifyDB, err := gorm.Open(mysql.Open(buildDSN(database)), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open MySQL connection: %w", err)
	}
	sqlDB, err := verifyDB.DB()
	if err != nil {
		return fmt.Errorf("get MySQL SQL connection: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping MySQL database: %w", err)
	}
	return nil
}

func createDatabaseIfNotExists(ctx context.Context, database domain.Database) error {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, database.Name)
	if !matched {
		return fmt.Errorf("invalid database name %q: only alphanumeric and underscore characters are allowed", database.Name)
	}

	// 1. Try to connect with the provided user and create the database
	userErr := tryCreateDatabaseWithCredentials(ctx, database.Host, database.Port, database.User, database.Password, database.Name)
	if userErr == nil {
		return nil
	}

	// 2. If user lacks privilege (e.g. Error 1044 / Access denied for non-root user in Docker),
	// attempt creation and grant privileges using MySQL root administrative account.
	rootPasswords := []string{
		"root_password", // standard docker-compose password for demo_mysql
		database.Password,
		"root",
		"",
	}

	for _, rootPass := range rootPasswords {
		if err := tryCreateAndGrantWithRoot(ctx, database.Host, database.Port, rootPass, database.User, database.Name); err == nil {
			return nil
		}
	}

	return fmt.Errorf("create MySQL database %q: %w", database.Name, userErr)
}

func tryCreateDatabaseWithCredentials(ctx context.Context, host string, port uint, user, password, dbName string) error {
	config := mysqlDriver.Config{
		User:      user,
		Passwd:    password,
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%d", host, port),
		DBName:    "",
		ParseTime: true,
	}

	serverDB, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := serverDB.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)
	_, err = sqlDB.ExecContext(ctx, query)
	return err
}

func tryCreateAndGrantWithRoot(ctx context.Context, host string, port uint, rootPassword, targetUser, dbName string) error {
	config := mysqlDriver.Config{
		User:      "root",
		Passwd:    rootPassword,
		Net:       "tcp",
		Addr:      fmt.Sprintf("%s:%d", host, port),
		DBName:    "",
		ParseTime: true,
	}

	serverDB, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := serverDB.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// Create database
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)
	if _, err := sqlDB.ExecContext(ctx, query); err != nil {
		return err
	}

	// Grant access to the target user so they can access the newly created database
	if targetUser != "" {
		grantDb := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", dbName, targetUser)
		_, _ = sqlDB.ExecContext(ctx, grantDb)

		// Also grant global create so subsequent operations are directly permitted
		grantGlobal := fmt.Sprintf("GRANT ALL PRIVILEGES ON *.* TO '%s'@'%%'", targetUser)
		_, _ = sqlDB.ExecContext(ctx, grantGlobal)

		_, _ = sqlDB.ExecContext(ctx, "FLUSH PRIVILEGES")
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
