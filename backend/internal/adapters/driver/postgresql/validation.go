package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

func (r *ExecutorRepository) ValidateMigrationSQL(ctx context.Context, db domain.Database, script string) error {
	target, err := r.poolManager.Get(ctx, db)
	if err != nil {
		return err
	}
	pool, err := target.DB()
	if err != nil {
		return err
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		pg := driverConn.(*stdlib.Conn).Conn().PgConn()
		return validateStatements(ctx, pg, script)
	})
}

func validateStatements(ctx context.Context, conn *pgconn.PgConn, script string) error {
	statements := sqlutil.SplitStatements(script)
	if len(statements) == 0 {
		return domain.ErrInvalidMigrationInput
	}
	for i, statement := range statements {
		words := strings.Fields(strings.ToUpper(statement))
		switch words[0] {
		case "BEGIN", "START", "COMMIT", "END", "ROLLBACK", "ABORT", "SAVEPOINT", "RELEASE", "PREPARE", "COPY":
			return fmt.Errorf("%w: unsupported SQL statement %d", domain.ErrInvalidMigrationInput, i+1)
		}
		if _, err := conn.Prepare(ctx, "", statement, nil); err != nil {
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) {
				return err
			}
			switch pgErr.Code {
			case "42P01", "42703", "42704", "42883", "3F000":
				continue
			default:
				return fmt.Errorf("%w: statement %d: %s", domain.ErrInvalidMigrationInput, i+1, pgErr.Message)
			}
		}
	}
	return nil
}
