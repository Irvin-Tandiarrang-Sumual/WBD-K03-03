package sqlutil_test

import (
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

func TestSplitStatements_ScriptWithMultipleStatementsAndComments_SplitsCorrectly(t *testing.T) {
	// Arrange
	sqlScript := `
		-- This is a comment;
		CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(50));
		/* Block comment with ; semicolon */
		INSERT INTO users (name) VALUES ('John; Doe');
		SELECT * FROM users;
	`

	// Act
	statements := sqlutil.SplitStatements(sqlScript)

	// Assert
	if len(statements) != 3 {
		t.Fatalf("expected 3 statements, got %d: %#v", len(statements), statements)
	}
	if statements[0] != "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(50))" {
		t.Fatalf("unexpected statement 0: %q", statements[0])
	}
	if statements[1] != "INSERT INTO users (name) VALUES ('John; Doe')" {
		t.Fatalf("unexpected statement 1: %q", statements[1])
	}
	if statements[2] != "SELECT * FROM users" {
		t.Fatalf("unexpected statement 2: %q", statements[2])
	}
}

func TestSplitValues_CommaSeparatedTokensWithQuotesAndParentheses_PreservesNestedStructures(t *testing.T) {
	// Arrange
	input := "1, 'hello, world', (SELECT count(*) FROM t), TRUE"
	expected := []string{"1", "'hello, world'", "(SELECT count(*) FROM t)", "TRUE"}

	// Act
	values := sqlutil.SplitValues(input)

	// Assert
	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("got %#v, want %#v", values, expected)
	}
}

func TestSplitValueGroups_ValidRowTuples_ReturnsIndividualTuples(t *testing.T) {
	// Arrange
	input := "(1, 'alice'), (2, 'bob''s')"
	expected := []string{"1, 'alice'", "2, 'bob''s'"}

	// Act
	groups, ok := sqlutil.SplitValueGroups(input)

	// Assert
	if !ok {
		t.Fatal("expected SplitValueGroups to succeed, got ok=false")
	}
	if !reflect.DeepEqual(groups, expected) {
		t.Fatalf("got %#v, want %#v", groups, expected)
	}
}

func TestSplitValueGroups_UnbalancedOrMalformedInput_ReturnsFalse(t *testing.T) {
	malformedInputs := []string{
		"(1, 'unclosed quote)",
		"(1, 2",
		"1, 2)",
		"invalid characters outside (1, 2)",
	}

	for _, input := range malformedInputs {
		t.Run("input="+input, func(t *testing.T) {
			// Arrange
			malformedStr := input

			// Act
			_, ok := sqlutil.SplitValueGroups(malformedStr)

			// Assert
			if ok {
				t.Fatalf("expected malformed input %q to return ok=false", malformedStr)
			}
		})
	}
}

func TestParseAssignments_ColumnValuePairs_ReturnsMap(t *testing.T) {
	// Arrange
	input := "name = 'John Doe', age = 30, active = TRUE"
	expected := map[string]string{
		"name":   "'John Doe'",
		"age":    "30",
		"active": "TRUE",
	}

	// Act
	assignments := sqlutil.ParseAssignments(input)

	// Assert
	if !reflect.DeepEqual(assignments, expected) {
		t.Fatalf("got assignments %+v, want %+v", assignments, expected)
	}
}

func TestFormatSQLValue_VariousTypes_ReturnsCorrectSQLString(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		{name: "nil value", input: nil, expected: "NULL"},
		{name: "string with single quotes", input: "it's fine", expected: "'it''s fine'"},
		{name: "byte slice", input: []byte("raw"), expected: "'raw'"},
		{name: "boolean true", input: true, expected: "TRUE"},
		{name: "boolean false", input: false, expected: "FALSE"},
		{name: "integer", input: 42, expected: "42"},
		{name: "float", input: 3.14, expected: "3.14"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			inputValue := tc.input
			expectedString := tc.expected

			// Act
			result := sqlutil.FormatSQLValue(inputValue)

			// Assert
			if result != expectedString {
				t.Fatalf("FormatSQLValue(%v) = %q, want %q", inputValue, result, expectedString)
			}
		})
	}
}

func TestReverseStrings_SliceOfStrings_ReversesInPlace(t *testing.T) {
	// Arrange
	items := []string{"first", "second", "third"}
	expected := []string{"third", "second", "first"}

	// Act
	sqlutil.ReverseStrings(items)

	// Assert
	if !reflect.DeepEqual(items, expected) {
		t.Fatalf("got reversed slice %+v, want %+v", items, expected)
	}
}

func TestIsWhitespace_StandardWhitespaceBytes_ReturnsTrue(t *testing.T) {
	whitespaceChars := []byte{' ', '\t', '\n', '\r'}

	for _, ch := range whitespaceChars {
		t.Run("char="+string(ch), func(t *testing.T) {
			// Arrange
			charByte := ch

			// Act
			isWhitespace := sqlutil.IsWhitespace(charByte)

			// Assert
			if !isWhitespace {
				t.Fatalf("expected byte %q to be considered whitespace, got false", charByte)
			}
		})
	}
}

func TestIsWhitespace_NonWhitespaceCharacters_ReturnsFalse(t *testing.T) {
	nonWhitespaceChars := []byte{'a', 'Z', '1', ';', '-', '*'}

	for _, ch := range nonWhitespaceChars {
		t.Run("char="+string(ch), func(t *testing.T) {
			// Arrange
			charByte := ch

			// Act
			isWhitespace := sqlutil.IsWhitespace(charByte)

			// Assert
			if isWhitespace {
				t.Fatalf("expected byte %q NOT to be considered whitespace, got true", charByte)
			}
		})
	}
}
