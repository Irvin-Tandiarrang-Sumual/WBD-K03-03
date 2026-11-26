package sqlutil_test

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

func TestInverseRename_ColumnRename_ReturnsInvertedStatement(t *testing.T) {
	// Arrange
	input := "ALTER TABLE users RENAME COLUMN email TO user_email"
	expected := "ALTER TABLE users RENAME COLUMN user_email TO email"

	// Act
	got, ok := sqlutil.InverseRename(input)

	// Assert
	if !ok || got != expected {
		t.Fatalf("expected (%q, true), got (%q, %t)", expected, got, ok)
	}
}

func TestInverseRename_TableRenameStandard_ReturnsInvertedStatement(t *testing.T) {
	// Arrange
	input := "ALTER TABLE old_users RENAME TO new_users"
	expected := "ALTER TABLE new_users RENAME TO old_users"

	// Act
	got, ok := sqlutil.InverseRename(input)

	// Assert
	if !ok || got != expected {
		t.Fatalf("expected (%q, true), got (%q, %t)", expected, got, ok)
	}
}

func TestInverseRename_TableRenameWithSchema_PreservesSchemaPrefix(t *testing.T) {
	// Arrange
	input := "ALTER TABLE public.old_users RENAME TO new_users"
	expected := "ALTER TABLE public.new_users RENAME TO old_users"

	// Act
	got, ok := sqlutil.InverseRename(input)

	// Assert
	if !ok || got != expected {
		t.Fatalf("expected (%q, true), got (%q, %t)", expected, got, ok)
	}
}

func TestInverseRename_MySQLRenameTable_ReturnsInvertedStatement(t *testing.T) {
	// Arrange
	input := "RENAME TABLE old_tbl TO new_tbl"
	expected := "RENAME TABLE new_tbl TO old_tbl"

	// Act
	got, ok := sqlutil.InverseRename(input)

	// Assert
	if !ok || got != expected {
		t.Fatalf("expected (%q, true), got (%q, %t)", expected, got, ok)
	}
}

func TestInverseRename_UnrelatedStatement_ReturnsFalse(t *testing.T) {
	// Arrange
	unrelated := []string{
		"CREATE TABLE users (id INT)",
		"DROP TABLE users",
		"ALTER TABLE users ADD COLUMN age INT",
		"SELECT * FROM users",
	}

	// Act & Assert
	for _, sql := range unrelated {
		got, ok := sqlutil.InverseRename(sql)
		if ok || got != "" {
			t.Fatalf("expected empty and false for %q, got (%q, %t)", sql, got, ok)
		}
	}
}
