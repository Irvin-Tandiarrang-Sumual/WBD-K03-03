package mysql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

var (
	createTableRegex     = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_.]+)`)
	createIndexRegex     = regexp.MustCompile(`(?i)^CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)`)
	mysqlIndexTableRegex = regexp.MustCompile(`(?i)\s+ON\s+([a-zA-Z0-9_.]+)\s*\(`)
	alterAddColumnRegex  = regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+([a-zA-Z0-9_.]+)\s+ADD\s+(?:COLUMN\s+)?([a-zA-Z0-9_]+)\s+`)
	insertRegex          = regexp.MustCompile(`(?is)^INSERT\s+INTO\s+([a-zA-Z0-9_.]+)\s*\(([^)]+)\)\s*VALUES\s*(.+)$`)
	deleteRegex          = regexp.MustCompile(`(?is)^DELETE\s+FROM\s+([a-zA-Z0-9_.]+)\s+WHERE\s+(.+)$`)
	updateRegex          = regexp.MustCompile(`(?is)^UPDATE\s+([a-zA-Z0-9_.]+)\s+SET\s+(.+?)\s+WHERE\s+(.+)$`)
)

type SQLDialect struct {
	executor ports.DatabaseExecutorRepository
}

var _ ports.SQLDialect = (*SQLDialect)(nil)

func NewSQLDialect(executor ports.DatabaseExecutorRepository) *SQLDialect {
	return &SQLDialect{executor: executor}
}

func (d *SQLDialect) GenerateSQLDown(ctx context.Context, database domain.Database, sqlUp string) (string, error) {
	statements := sqlutil.SplitStatements(sqlUp)
	if len(statements) == 0 {
		return "", domain.ErrInvalidMigrationInput
	}

	downStatements := make([]string, 0, len(statements))
	for _, statement := range statements {
		down, err := d.generateSingleDown(ctx, database, statement)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(down) != "" {
			downStatements = append(downStatements, down)
		}
	}

	sqlutil.ReverseStrings(downStatements)
	return strings.Join(downStatements, ";\n"), nil
}

func (d *SQLDialect) generateSingleDown(ctx context.Context, database domain.Database, rawStatement string) (string, error) {
	statement := strings.TrimSpace(strings.TrimSuffix(rawStatement, ";"))
	lower := strings.ToLower(statement)
	if statement == "" || strings.HasPrefix(lower, "select ") {
		return "", domain.ErrInvalidMigrationInput
	}

	switch {
	case strings.HasPrefix(lower, "create table "):
		return generateCreateTableDown(statement)
	case strings.HasPrefix(lower, "create index "), strings.HasPrefix(lower, "create unique index "):
		return generateCreateIndexDown(statement)
	case strings.HasPrefix(lower, "alter table "):
		return generateAlterTableDown(statement)
	case strings.HasPrefix(lower, "insert into "):
		return generateInsertDown(statement)
	case strings.HasPrefix(lower, "delete from "):
		return d.generateDeleteDown(ctx, database, statement)
	case strings.HasPrefix(lower, "update "):
		return d.generateUpdateDown(ctx, database, statement)
	default:
		return "", domain.ErrSQLDownGenerationUnsupported
	}
}

func generateCreateTableDown(statement string) (string, error) {
	matches := createTableRegex.FindStringSubmatch(statement)
	if len(matches) < 2 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	tableName := strings.TrimSpace(matches[1])
	if !sqlutil.IsValidIdentifier(tableName) {
		return "", domain.ErrInvalidMigrationInput
	}
	return "DROP TABLE " + tableName, nil
}

func generateCreateIndexDown(statement string) (string, error) {
	indexMatches := createIndexRegex.FindStringSubmatch(statement)
	tableMatches := mysqlIndexTableRegex.FindStringSubmatch(statement)
	if len(indexMatches) < 2 || len(tableMatches) < 2 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	indexName := strings.TrimSpace(indexMatches[1])
	tableName := strings.TrimSpace(tableMatches[1])
	if !sqlutil.IsValidIdentifier(indexName) || !sqlutil.IsValidIdentifier(tableName) {
		return "", domain.ErrInvalidMigrationInput
	}
	return fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName), nil
}

func generateAlterTableDown(statement string) (string, error) {
	matches := alterAddColumnRegex.FindStringSubmatch(statement)
	if len(matches) != 3 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}
	tableName := strings.TrimSpace(matches[1])
	columnName := strings.TrimSpace(matches[2])
	if !sqlutil.IsValidIdentifier(tableName) || !sqlutil.IsValidIdentifier(columnName) {
		return "", domain.ErrInvalidMigrationInput
	}
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName), nil
}

func generateInsertDown(statement string) (string, error) {
	matches := insertRegex.FindStringSubmatch(statement)
	if len(matches) != 4 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}

	tableName := strings.TrimSpace(matches[1])
	if !sqlutil.IsValidIdentifier(tableName) {
		return "", domain.ErrInvalidMigrationInput
	}

	columns := sqlutil.SplitValues(matches[2])
	valueGroups, ok := sqlutil.SplitValueGroups(matches[3])
	if !ok || len(valueGroups) == 0 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}

	for _, col := range columns {
		if !sqlutil.IsValidIdentifier(strings.TrimSpace(col)) {
			return "", domain.ErrInvalidMigrationInput
		}
	}

	rowPredicates := make([]string, 0, len(valueGroups))
	for _, group := range valueGroups {
		values := sqlutil.SplitValues(group)
		if len(columns) != len(values) {
			return "", domain.ErrSQLDownGenerationUnsupported
		}

		predicates := make([]string, 0, len(columns))
		for i, col := range columns {
			colName := strings.TrimSpace(col)
			val := strings.TrimSpace(values[i])
			if !sqlutil.IsValidLiteralValue(val) {
				return "", domain.ErrInvalidMigrationInput
			}
			if strings.EqualFold(val, "null") {
				predicates = append(predicates, colName+" IS NULL")
			} else {
				predicates = append(predicates, colName+" = "+val)
			}
		}
		rowPredicates = append(rowPredicates, strings.Join(predicates, " AND "))
	}

	if len(rowPredicates) == 1 {
		return fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, rowPredicates[0]), nil
	}

	for i, predicate := range rowPredicates {
		rowPredicates[i] = "(" + predicate + ")"
	}
	return fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, strings.Join(rowPredicates, " OR ")), nil
}

func (d *SQLDialect) generateDeleteDown(ctx context.Context, database domain.Database, statement string) (string, error) {
	matches := deleteRegex.FindStringSubmatch(statement)
	if len(matches) != 3 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}

	tableName := strings.TrimSpace(matches[1])
	whereClause := strings.TrimSpace(matches[2])

	if !sqlutil.IsValidIdentifier(tableName) {
		return "", domain.ErrInvalidMigrationInput
	}

	query := "SELECT * FROM " + tableName + " WHERE " + whereClause
	isReadOnly, err := sqlutil.IsReadOnlyQuery(query)
	if err != nil || !isReadOnly {
		return "", domain.ErrInvalidMigrationInput
	}

	result, err := d.executor.ExecuteQuery(ctx, database, query)
	if err != nil {
		return "", err
	}

	return generateInsertStatements(tableName, result), nil
}

func (d *SQLDialect) generateUpdateDown(ctx context.Context, database domain.Database, statement string) (string, error) {
	matches := updateRegex.FindStringSubmatch(statement)
	if len(matches) != 4 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}

	tableName := strings.TrimSpace(matches[1])
	assignments := sqlutil.ParseAssignments(matches[2])
	whereClause := strings.TrimSpace(matches[3])

	if len(assignments) == 0 || !sqlutil.IsValidIdentifier(tableName) {
		return "", domain.ErrSQLDownGenerationUnsupported
	}

	for col, val := range assignments {
		if !sqlutil.IsValidIdentifier(col) || !sqlutil.IsValidLiteralValue(val) {
			return "", domain.ErrInvalidMigrationInput
		}
	}

	query := "SELECT * FROM " + tableName + " WHERE " + whereClause
	isReadOnly, err := sqlutil.IsReadOnlyQuery(query)
	if err != nil || !isReadOnly {
		return "", domain.ErrInvalidMigrationInput
	}

	result, err := d.executor.ExecuteQuery(ctx, database, query)
	if err != nil {
		return "", err
	}

	statements := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		updateStmt, err := buildInverseUpdateRow(tableName, row, assignments)
		if err != nil {
			return "", err
		}
		statements = append(statements, updateStmt)
	}

	return strings.Join(statements, ";\n"), nil
}

func buildInverseUpdateRow(tableName string, row map[string]any, assignments map[string]string) (string, error) {
	setParts := make([]string, 0, len(assignments))
	whereParts := make([]string, 0, len(row))

	for column, oldValue := range row {
		if assignment, exists := assignments[column]; exists {
			setParts = append(setParts, column+" = "+sqlutil.FormatSQLValue(oldValue))
			whereParts = append(whereParts, column+" = "+assignment)
		} else {
			if oldValue == nil {
				whereParts = append(whereParts, column+" IS NULL")
			} else {
				whereParts = append(whereParts, column+" = "+sqlutil.FormatSQLValue(oldValue))
			}
		}
	}

	if len(setParts) == 0 {
		return "", domain.ErrSQLDownGenerationUnsupported
	}

	return fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setParts, ", "),
		strings.Join(whereParts, " AND "),
	), nil
}

func generateInsertStatements(table string, result *ports.DatabaseQueryResult) string {
	statements := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		columns := make([]string, 0, len(result.Columns))
		values := make([]string, 0, len(result.Columns))
		for _, column := range result.Columns {
			columns = append(columns, column)
			values = append(values, sqlutil.FormatSQLValue(row[column]))
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
