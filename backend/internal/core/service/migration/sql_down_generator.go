package migration

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func generateSQLDown(
	ctx context.Context,
	executor ports.DatabaseExecutor,
	database domain.Database,
	sqlUp string,
) (string, error) {
	statement := strings.TrimSpace(strings.TrimSuffix(sqlUp, ";"))
	lowerStatement := strings.ToLower(statement)
	if statement == "" || strings.HasPrefix(lowerStatement, "select ") {
		return "", domain.ErrInvalidMigrationInput
	}

	switch {
	case strings.HasPrefix(lowerStatement, "create table "):
		tableName, ok := captureIdentifier(statement, `(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_.]+)`)
		if !ok {
			return "", domain.ErrSQLDownGenerationUnsupported
		}
		return "DROP TABLE " + tableName, nil
	case strings.HasPrefix(lowerStatement, "create index "),
		strings.HasPrefix(lowerStatement, "create unique index "):
		indexName, ok := captureIdentifier(statement, `(?i)^CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)`)
		if !ok {
			return "", domain.ErrSQLDownGenerationUnsupported
		}
		if database.Type == "mysql" {
			tableName, tableOK := captureIdentifier(statement, `(?i)\s+ON\s+([a-zA-Z0-9_.]+)\s*\(`)
			if !tableOK {
				return "", domain.ErrSQLDownGenerationUnsupported
			}
			return fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName), nil
		}
		return "DROP INDEX " + indexName, nil
	case strings.HasPrefix(lowerStatement, "alter table "):
		tableName, columnName, ok := parseAddColumn(statement)
		if !ok {
			return "", domain.ErrSQLDownGenerationUnsupported
		}
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName), nil
	case strings.HasPrefix(lowerStatement, "insert into "):
		return generateInsertDown(statement)
	case strings.HasPrefix(lowerStatement, "delete from "):
		return generateDeleteDown(ctx, executor, database, statement)
	case strings.HasPrefix(lowerStatement, "update "):
		return generateUpdateDown(ctx, executor, database, statement)
	default:
		return "", domain.ErrSQLDownGenerationUnsupported
	}
}

func generateInsertDown(statement string) (string, error) {
	re := regexp.MustCompile(`(?is)^INSERT\s+INTO\s+([a-zA-Z0-9_.]+)\s*\(([^)]+)\)\s*VALUES\s*\((.+)\)$`)
	matches := re.FindStringSubmatch(statement)
	if len(matches) != 4 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	columns := splitSQLValues(matches[2])
	values := splitSQLValues(matches[3])
	if len(columns) != len(values) {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	predicates := make([]string, 0, len(columns))
	for index, column := range columns {
		predicates = append(predicates, strings.TrimSpace(column)+" = "+strings.TrimSpace(values[index]))
	}
	return fmt.Sprintf("DELETE FROM %s WHERE %s", matches[1], strings.Join(predicates, " AND ")), nil
}

func generateDeleteDown(ctx context.Context, executor ports.DatabaseExecutor, database domain.Database, statement string) (string, error) {
	re := regexp.MustCompile(`(?is)^DELETE\s+FROM\s+([a-zA-Z0-9_.]+)\s+WHERE\s+(.+)$`)
	matches := re.FindStringSubmatch(statement)
	if len(matches) != 3 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	result, err := executor.ExecuteQuery(ctx, database, "SELECT * FROM "+matches[1]+" WHERE "+matches[2])
	if err != nil {
		return "", err
	}
	return generateInsertStatements(matches[1], result), nil
}

func generateUpdateDown(ctx context.Context, executor ports.DatabaseExecutor, database domain.Database, statement string) (string, error) {
	re := regexp.MustCompile(`(?is)^UPDATE\s+([a-zA-Z0-9_.]+)\s+SET\s+(.+)\s+WHERE\s+(.+)$`)
	matches := re.FindStringSubmatch(statement)
	if len(matches) != 4 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	assignments := parseAssignments(matches[2])
	if len(assignments) == 0 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	result, err := executor.ExecuteQuery(ctx, database, "SELECT * FROM "+matches[1]+" WHERE "+matches[3])
	if err != nil {
		return "", err
	}
	statements := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		setParts := make([]string, 0, len(assignments))
		whereParts := make([]string, 0, len(row))
		for column, oldValue := range row {
			newValue := oldValue
			if assignment, exists := assignments[column]; exists {
				newValue = assignment
				setParts = append(setParts, column+" = "+formatSQLValue(oldValue))
			}
			whereParts = append(whereParts, column+" = "+formatSQLValue(newValue))
		}
		if len(setParts) == 0 {
			return "", domain.ErrSQLDownGenerationUnsupported
		}
		statements = append(statements, fmt.Sprintf(
			"UPDATE %s SET %s WHERE %s",
			matches[1],
			strings.Join(setParts, ", "),
			strings.Join(whereParts, " AND "),
		))
	}
	return strings.Join(statements, ";\n"), nil
}

func generateInsertStatements(table string, result *ports.DatabaseQueryResult) string {
	statements := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		columns := make([]string, 0, len(result.Columns))
		values := make([]string, 0, len(result.Columns))
		for _, column := range result.Columns {
			columns = append(columns, column)
			values = append(values, formatSQLValue(row[column]))
		}
		statements = append(statements, fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			table,
			strings.Join(columns, ", "),
			strings.Join(values, ", "),
		))
	}
	return strings.Join(statements, ";\n")
}

func parseAssignments(input string) map[string]string {
	assignments := make(map[string]string)
	for _, assignment := range splitSQLValues(input) {
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) == 2 {
			assignments[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return assignments
}

func parseAddColumn(statement string) (string, string, bool) {
	re := regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+([a-zA-Z0-9_.]+)\s+ADD\s+(?:COLUMN\s+)?([a-zA-Z0-9_]+)\s+`)
	matches := re.FindStringSubmatch(statement)
	returnValue := len(matches) == 3
	if !returnValue {
		return "", "", false
	}
	return matches[1], matches[2], true
}

func captureIdentifier(statement string, pattern string) (string, bool) {
	matches := regexp.MustCompile(pattern).FindStringSubmatch(statement)
	returnValue := len(matches) == 2
	if !returnValue {
		return "", false
	}
	return matches[1], true
}

func splitSQLValues(input string) []string {
	values := make([]string, 0)
	start := 0
	depth := 0
	quoted := false
	for index, character := range input {
		switch character {
		case '\'':
			quoted = !quoted
		case '(':
			if !quoted {
				depth++
			}
		case ')':
			if !quoted {
				depth--
			}
		case ',':
			if !quoted && depth == 0 {
				values = append(values, strings.TrimSpace(input[start:index]))
				start = index + 1
			}
		}
	}
	values = append(values, strings.TrimSpace(input[start:]))
	return values
}

func formatSQLValue(value any) string {
	if value == nil {
		return "NULL"
	}
	switch typedValue := value.(type) {
	case []byte:
		return "'" + strings.ReplaceAll(string(typedValue), "'", "''") + "'"
	case string:
		return "'" + strings.ReplaceAll(typedValue, "'", "''") + "'"
	case bool:
		if typedValue {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprint(typedValue)
	}
}
