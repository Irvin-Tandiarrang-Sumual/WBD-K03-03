package sqlutil_test

import (
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

func TestIsReadOnlyQuery_ReadOnlyStatements_ReturnsTrue(t *testing.T) {
	testCases := []struct {
		name  string
		query string
	}{
		{name: "standard select", query: "SELECT * FROM users"},
		{name: "select count aggregate", query: "SELECT count(*) FROM users"},
		{name: "read-only common table expression", query: "WITH cte AS (SELECT 1) SELECT * FROM cte"},
		{name: "semicolon inside string literal", query: "SELECT * FROM logs WHERE message = 'error; please retry'"},
		{name: "comment containing semicolon", query: "SELECT 1 /* comment; */ FROM users"},
		{name: "column name containing keyword substring", query: "SELECT updated_at, is_deleted FROM users"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			query := tc.query

			// Act
			isReadOnly, err := sqlutil.IsReadOnlyQuery(query)

			// Assert
			if err != nil {
				t.Fatalf("unexpected error for query %q: %v", query, err)
			}
			if !isReadOnly {
				t.Fatalf("expected query %q to be read-only, got false", query)
			}
		})
	}
}

func TestIsReadOnlyQuery_MutationOrInvalidStatements_ReturnsFalseOrError(t *testing.T) {
	testCases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{name: "blank query", query: "   ", wantErr: false},
		{name: "update query", query: "UPDATE users SET active = true", wantErr: false},
		{name: "delete query", query: "DELETE FROM users WHERE id = 1", wantErr: false},
		{name: "insert query", query: "INSERT INTO users (name) VALUES ('a')", wantErr: false},
		{name: "stacked multiple statements", query: "SELECT 1; SELECT 2", wantErr: false},
		{name: "writable cte with delete", query: "WITH d AS (DELETE FROM users RETURNING *) SELECT * FROM d", wantErr: false},
		{name: "writable cte with update", query: "WITH u AS (UPDATE users SET role = 'admin' RETURNING *) SELECT * FROM u", wantErr: false},
		{name: "unclosed string literal syntax error", query: "SELECT * FROM users WHERE name = 'abc", wantErr: true},
		{name: "unclosed block comment syntax error", query: "SELECT 1 /* comment", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			query := tc.query

			// Act
			isReadOnly, err := sqlutil.IsReadOnlyQuery(query)

			// Assert
			if (err != nil) != tc.wantErr {
				t.Fatalf("query %q: got error = %v, wantErr = %v", query, err, tc.wantErr)
			}
			if isReadOnly {
				t.Fatalf("expected query %q NOT to be read-only, got true", query)
			}
		})
	}
}

func TestContainsWord_ExactWordPresent_ReturnsTrue(t *testing.T) {
	// Arrange
	text := "select delete from x"
	word := "delete"

	// Act
	found := sqlutil.ContainsWord(text, word)

	// Assert
	if !found {
		t.Fatal("expected exact word 'delete' to match, got false")
	}
}

func TestContainsWord_SubstringInsideOtherIdentifier_ReturnsFalse(t *testing.T) {
	// Arrange
	text := "select deleted, is_deleted from x"
	word := "delete"

	// Act
	found := sqlutil.ContainsWord(text, word)

	// Assert
	if found {
		t.Fatal("did not expect 'delete' to match substring inside other identifiers, got true")
	}
}

func TestContainsWord_WordPrecededByParenthesis_ReturnsTrue(t *testing.T) {
	// Arrange
	text := "with d as (delete from x)"
	word := "delete"

	// Act
	found := sqlutil.ContainsWord(text, word)

	// Assert
	if !found {
		t.Fatal("expected 'delete' to match when preceded by parenthesis, got false")
	}
}

func TestIsValidIdentifier_ValidSQLIdentifiers_ReturnsTrue(t *testing.T) {
	validIdentifiers := []string{
		"users", "public.users", `"users"`, "`users`", "user_accounts", "u1", `"schema"."table"`,
	}

	for _, id := range validIdentifiers {
		t.Run("valid="+id, func(t *testing.T) {
			// Arrange
			identifier := id

			// Act
			isValid := sqlutil.IsValidIdentifier(identifier)

			// Assert
			if !isValid {
				t.Fatalf("expected %q to be considered a valid identifier, got false", identifier)
			}
		})
	}
}

func TestIsValidIdentifier_InvalidOrInjectedCharacters_ReturnsFalse(t *testing.T) {
	invalidIdentifiers := []string{
		"", "users; drop table x", "users (id", "users--", "123users", "a.b.c", "users;--",
	}

	for _, id := range invalidIdentifiers {
		t.Run("invalid="+id, func(t *testing.T) {
			// Arrange
			identifier := id

			// Act
			isValid := sqlutil.IsValidIdentifier(identifier)

			// Assert
			if isValid {
				t.Fatalf("expected %q to be considered an invalid identifier, got true", identifier)
			}
		})
	}
}

func TestIsValidLiteralValue_ValidSQLLiterals_ReturnsTrue(t *testing.T) {
	validLiterals := []string{
		"1", "-42", "3.14", "'hello'", "'Ada;Grace'", "'it''s fine'", "TRUE", "false", "NULL", "null",
	}

	for _, lit := range validLiterals {
		t.Run("valid="+lit, func(t *testing.T) {
			// Arrange
			literal := lit

			// Act
			isValid := sqlutil.IsValidLiteralValue(literal)

			// Assert
			if !isValid {
				t.Fatalf("expected %q to be considered a valid literal value, got false", literal)
			}
		})
	}
}

func TestIsValidLiteralValue_UnsafeOrInjectedExpressions_ReturnsFalse(t *testing.T) {
	invalidLiterals := []string{
		"", "'unclosed", "'Ada' OR 1=1", "1; DROP TABLE users", "(SELECT 1)",
	}

	for _, lit := range invalidLiterals {
		t.Run("invalid="+lit, func(t *testing.T) {
			// Arrange
			literal := lit

			// Act
			isValid := sqlutil.IsValidLiteralValue(literal)

			// Assert
			if isValid {
				t.Fatalf("expected %q to be considered an invalid literal value, got true", literal)
			}
		})
	}
}
