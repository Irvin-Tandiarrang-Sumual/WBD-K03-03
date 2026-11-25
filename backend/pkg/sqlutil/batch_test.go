package sqlutil_test

import (
	"errors"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

func TestValidateBatchDML_ValidSingleUpdate_ReturnsTableName(t *testing.T) {
	// Arrange
	sql := "UPDATE users SET status = 'active' WHERE id > 0;"

	// Act
	tbl, err := sqlutil.ValidateBatchDML(sql)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tbl != "users" {
		t.Fatalf("expected table users, got %q", tbl)
	}
}

func TestValidateBatchDML_ValidMultipleStatementsSameTable_ReturnsTableName(t *testing.T) {
	// Arrange
	sql := "UPDATE users SET status = 'active' WHERE id > 0; DELETE FROM users WHERE status = 'banned';"

	// Act
	tbl, err := sqlutil.ValidateBatchDML(sql)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tbl != "users" {
		t.Fatalf("expected table users, got %q", tbl)
	}
}

func TestValidateBatchDML_ContainsDDL_ReturnsErrBatchContainsDDL(t *testing.T) {
	// Arrange
	testCases := []struct {
		name string
		sql  string
	}{
		{"create table", "UPDATE users SET status = 'active'; CREATE TABLE secret (id int);"},
		{"alter table", "ALTER TABLE users ADD COLUMN phone varchar(20);"},
		{"drop table", "DROP TABLE users;"},
		{"truncate table", "TRUNCATE TABLE users;"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			_, err := sqlutil.ValidateBatchDML(tc.sql)

			// Assert
			if !errors.Is(err, sqlutil.ErrBatchContainsDDL) {
				t.Fatalf("expected ErrBatchContainsDDL, got %v", err)
			}
		})
	}
}

func TestValidateBatchDML_MultipleUnrelatedTables_ReturnsErrBatchMultipleUnrelatedTables(t *testing.T) {
	// Arrange
	sql := "UPDATE users SET status = 'active' WHERE id > 0; UPDATE orders SET total = 0 WHERE id > 0;"

	// Act
	_, err := sqlutil.ValidateBatchDML(sql)

	// Assert
	if !errors.Is(err, sqlutil.ErrBatchMultipleUnrelatedTables) {
		t.Fatalf("expected ErrBatchMultipleUnrelatedTables, got %v", err)
	}
}
