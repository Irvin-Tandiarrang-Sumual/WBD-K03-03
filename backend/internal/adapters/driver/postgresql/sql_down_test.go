package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/testutil"
)

func TestSQLDialect_EmptyOrWhitespaceStatement_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, "   ;   ")

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || down != "" {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, down=%q", err, down)
	}
}

func TestSQLDialect_SelectQueryStatement_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, "SELECT * FROM users;")

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || down != "" {
		t.Fatalf("expected ErrInvalidMigrationInput, got err=%v, down=%q", err, down)
	}
}

func TestSQLDialect_CreateTableWithComplexSchemaAndConstraints_GeneratesDropTableStatement(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)
	sqlUp := `CREATE TABLE IF NOT EXISTS "store"."order_items" (
		order_id INT NOT NULL,
		item_id INT NOT NULL,
		price NUMERIC(10,2) CHECK (price > 0),
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (order_id, item_id),
		FOREIGN KEY (order_id) REFERENCES "store"."orders"(id) ON DELETE CASCADE
	);`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if err != nil || down != `DROP TABLE "store"."order_items"` {
		t.Fatalf("expected 'DROP TABLE \"store\".\"order_items\"', got err=%v, down=%q", err, down)
	}
}

func TestSQLDialect_CreateUniqueIndex_GeneratesDropIndexStatement(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)
	sqlUp := `CREATE UNIQUE INDEX "idx_users_email_tenant" ON "users" (email, tenant_id);`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	expectedDown := `DROP INDEX "idx_users_email_tenant"`
	if err != nil || down != expectedDown {
		t.Fatalf("expected %q, got err=%v, down=%q", expectedDown, err, down)
	}
}

func TestSQLDialect_AlterTableAddColumnWithComplexDefinition_GeneratesDropColumnStatement(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)
	sqlUp := `ALTER TABLE "store"."customers" ADD COLUMN "metadata" JSONB NULL;`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	expectedDown := `ALTER TABLE "store"."customers" DROP COLUMN "metadata"`
	if err != nil || down != expectedDown {
		t.Fatalf("expected %q, got err=%v, down=%q", expectedDown, err, down)
	}
}

func TestSQLDialect_InsertWithMultipleRowsEscapedQuotesAndNulls_GeneratesCompensatingDeleteStatement(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)
	sqlUp := `INSERT INTO "customers" (id, name, bio, balance, deleted_at) VALUES (1, 'O''Connor', 'Line 1\nLine 2', 250.75, NULL), (2, 'Smith', 'Simple bio', 0, '2026-01-01');`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	expectedDown := `DELETE FROM "customers" WHERE (id = 1 AND name = 'O''Connor' AND bio = 'Line 1\nLine 2' AND balance = 250.75 AND deleted_at IS NULL) OR (id = 2 AND name = 'Smith' AND bio = 'Simple bio' AND balance = 0 AND deleted_at = '2026-01-01')`
	if err != nil || down != expectedDown {
		t.Fatalf("expected %q, got err=%v, down=%q", expectedDown, err, down)
	}
}

func TestSQLDialect_DeleteWithNestedSubqueryInWhere_ExecutesReadOnlySelectAndGeneratesCompensatingInserts(t *testing.T) {
	// Arrange
	var capturedQuery string
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			capturedQuery = query
			return &ports.DatabaseQueryResult{
				Columns: []string{"id", "user_id", "total_amount", "notes"},
				Rows: []map[string]any{
					{
						"id":           101,
						"user_id":      5,
						"total_amount": 750.50,
						"notes":        "vip customer",
					},
					{
						"id":           102,
						"user_id":      9,
						"total_amount": 600.00,
						"notes":        nil,
					},
				},
			}, nil
		},
	}
	dialect := NewSQLDialect(executor)
	sqlUp := `DELETE FROM "orders" WHERE user_id IN (SELECT id FROM "users" WHERE status = 'inactive' AND age > 30) AND total_amount > 500.00;`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedCapturedQuery := `SELECT * FROM "orders" WHERE user_id IN (SELECT id FROM "users" WHERE status = 'inactive' AND age > 30) AND total_amount > 500.00`
	if capturedQuery != expectedCapturedQuery {
		t.Fatalf("expected captured query %q, got %q", expectedCapturedQuery, capturedQuery)
	}
	expectedDown := `INSERT INTO "orders" (id, user_id, total_amount, notes) VALUES (101, 5, 750.5, 'vip customer');` + "\n" + `INSERT INTO "orders" (id, user_id, total_amount, notes) VALUES (102, 9, 600, NULL)`
	if down != expectedDown {
		t.Fatalf("expected down statement %q, got %q", expectedDown, down)
	}
}

func TestSQLDialect_DeleteWithMutatingSubqueryInWhere_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	executor := &testutil.DatabaseExecutorRepository{}
	dialect := NewSQLDialect(executor)
	sqlUp := `DELETE FROM "orders" WHERE user_id IN (DELETE FROM "banned_users" WHERE id = 1 RETURNING id);`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || down != "" {
		t.Fatalf("expected ErrInvalidMigrationInput for mutating subquery in DELETE, got err=%v, down=%q", err, down)
	}
}

func TestSQLDialect_DeleteWhenExecutorReturnsError_SurfacesError(t *testing.T) {
	// Arrange
	executorErr := sql.ErrConnDone
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return nil, executorErr
		},
	}
	dialect := NewSQLDialect(executor)
	sqlUp := `DELETE FROM "orders" WHERE user_id = 42;`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if !errors.Is(err, executorErr) || down != "" {
		t.Fatalf("expected %v, got err=%v, down=%q", executorErr, err, down)
	}
}

func TestSQLDialect_UpdateWithNestedSubqueryInWhere_ExecutesReadOnlySelectAndGeneratesCompensatingUpdate(t *testing.T) {
	// Arrange
	var capturedQuery string
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			capturedQuery = query
			return &ports.DatabaseQueryResult{
				Columns: []string{"id", "price"},
				Rows: []map[string]any{
					{
						"id":    42,
						"price": 149.50,
					},
				},
			}, nil
		},
	}
	dialect := NewSQLDialect(executor)
	sqlUp := `UPDATE "products" SET price = 99.99 WHERE category_id IN (SELECT id FROM "categories" WHERE department = 'electronics');`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedCapturedQuery := `SELECT * FROM "products" WHERE category_id IN (SELECT id FROM "categories" WHERE department = 'electronics')`
	if capturedQuery != expectedCapturedQuery {
		t.Fatalf("expected captured query %q, got %q", expectedCapturedQuery, capturedQuery)
	}
	if !strings.HasPrefix(down, `UPDATE "products" SET price = 149.5 WHERE `) {
		t.Fatalf("expected compensating update restoring old price, got %q", down)
	}
	if !strings.Contains(down, "price = 99.99") || !strings.Contains(down, "id = 42") {
		t.Fatalf("expected WHERE clause to contain target price and ID predicates, got %q", down)
	}
}

func TestSQLDialect_UpdateWithMutatingSubqueryInWhere_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	executor := &testutil.DatabaseExecutorRepository{}
	dialect := NewSQLDialect(executor)
	sqlUp := `UPDATE "products" SET price = 10.00 WHERE category_id IN (DROP TABLE "categories");`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || down != "" {
		t.Fatalf("expected ErrInvalidMigrationInput for mutating subquery in UPDATE, got err=%v, down=%q", err, down)
	}
}

func TestSQLDialect_UpdateWhenExecutorReturnsError_SurfacesError(t *testing.T) {
	// Arrange
	executorErr := context.DeadlineExceeded
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return nil, executorErr
		},
	}
	dialect := NewSQLDialect(executor)
	sqlUp := `UPDATE "products" SET price = 20 WHERE id = 1;`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if !errors.Is(err, executorErr) || down != "" {
		t.Fatalf("expected %v, got err=%v, down=%q", executorErr, err, down)
	}
}

func TestSQLDialect_MultiStatementWithNestedQueriesAndDDL_ReversesCompensatingStatementsInProperOrder(t *testing.T) {
	// Arrange
	executor := &testutil.DatabaseExecutorRepository{
		ExecuteQueryFunc: func(ctx context.Context, database domain.Database, query string) (*ports.DatabaseQueryResult, error) {
			return &ports.DatabaseQueryResult{
				Columns: []string{"id", "user_id"},
				Rows: []map[string]any{
					{"id": 99, "user_id": 5},
				},
			}, nil
		},
	}
	dialect := NewSQLDialect(executor)
	sqlUp := `CREATE TABLE "audit_records" (id INT PRIMARY KEY, action VARCHAR(100));
INSERT INTO "audit_records" (id, action) VALUES (1, 'system_init');
DELETE FROM "temp_sessions" WHERE user_id IN (SELECT id FROM "users" WHERE is_banned = 1);`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	parts := strings.Split(down, ";\n")
	if len(parts) != 3 {
		t.Fatalf("expected 3 compensating statements, got %d (%v)", len(parts), parts)
	}
	if parts[0] != `INSERT INTO "temp_sessions" (id, user_id) VALUES (99, 5)` {
		t.Fatalf("expected first down statement to restore deleted temp_session, got %q", parts[0])
	}
	if parts[1] != `DELETE FROM "audit_records" WHERE id = 1 AND action = 'system_init'` {
		t.Fatalf("expected second down statement to delete inserted audit record, got %q", parts[1])
	}
	if parts[2] != `DROP TABLE "audit_records"` {
		t.Fatalf("expected third down statement to drop table, got %q", parts[2])
	}
}

func TestSQLDialect_UnsupportedDDLStatement_ReturnsErrSQLDownGenerationUnsupported(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)
	sqlUp := `TRUNCATE TABLE "legacy_users";`

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if !errors.Is(err, domain.ErrSQLDownGenerationUnsupported) || down != "" {
		t.Fatalf("expected ErrSQLDownGenerationUnsupported, got err=%v, down=%q", err, down)
	}
}

func TestSQLDialect_InvalidIdentifierInCreateTable_ReturnsInvalidMigrationInputError(t *testing.T) {
	// Arrange
	dialect := NewSQLDialect(nil)
	sqlUp := "CREATE TABLE 123invalid (id INT);"

	// Act
	down, err := dialect.GenerateSQLDown(context.Background(), domain.Database{}, sqlUp)

	// Assert
	if !errors.Is(err, domain.ErrInvalidMigrationInput) || down != "" {
		t.Fatalf("expected ErrInvalidMigrationInput for invalid identifier, got err=%v, down=%q", err, down)
	}
}
