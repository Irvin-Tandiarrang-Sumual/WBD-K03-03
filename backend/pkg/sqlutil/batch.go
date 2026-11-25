package sqlutil

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrBatchContainsDDL             = errors.New("Batch draft must only contain DML statements (INSERT, UPDATE, DELETE)")
	ErrBatchMultipleUnrelatedTables = errors.New("Batch draft statements must target the same table")
	ErrInvalidSQLStatement          = errors.New("Invalid SQL statement")
)

var (
	ddlRegex       = regexp.MustCompile(`(?i)\b(CREATE|DROP|ALTER|TRUNCATE|RENAME)\b`)
	updateTableReg = regexp.MustCompile(`(?i)^\s*UPDATE\s+([^\s\(\)]+)`)
	deleteTableReg = regexp.MustCompile(`(?i)^\s*DELETE\s+(?:FROM\s+)?([^\s\(\)]+)`)
	insertTableReg = regexp.MustCompile(`(?i)^\s*INSERT\s+(?:INTO\s+)?([^\s\(\)]+)`)
)

func ValidateBatchDML(sqlScript string) (string, error) {
	trimmed := strings.TrimSpace(sqlScript)
	if trimmed == "" {
		return "", ErrInvalidSQLStatement
	}

	statements := SplitStatements(trimmed)
	if len(statements) == 0 {
		return "", ErrInvalidSQLStatement
	}

	var targetTable string

	for _, stmt := range statements {
		sanitized, err := StripCommentsAndLiterals(stmt)
		if err != nil {
			return "", ErrInvalidSQLStatement
		}
		clean := strings.TrimSpace(sanitized)
		if clean == "" {
			continue
		}

		if ddlRegex.MatchString(clean) {
			return "", ErrBatchContainsDDL
		}

		var tbl string
		if matches := updateTableReg.FindStringSubmatch(clean); len(matches) > 1 {
			tbl = cleanIdentifier(matches[1])
		} else if matches := deleteTableReg.FindStringSubmatch(clean); len(matches) > 1 {
			tbl = cleanIdentifier(matches[1])
		} else if matches := insertTableReg.FindStringSubmatch(clean); len(matches) > 1 {
			tbl = cleanIdentifier(matches[1])
		} else {
			return "", ErrBatchContainsDDL
		}

		if tbl == "" {
			return "", ErrInvalidSQLStatement
		}

		if targetTable == "" {
			targetTable = tbl
		} else if !strings.EqualFold(targetTable, tbl) {
			return "", ErrBatchMultipleUnrelatedTables
		}
	}

	if targetTable == "" {
		return "", ErrInvalidSQLStatement
	}

	return targetTable, nil
}

func cleanIdentifier(ident string) string {
	ident = strings.TrimSpace(ident)
	ident = strings.Trim(ident, "`\"[]")
	if parts := strings.Split(ident, "."); len(parts) > 1 {
		ident = parts[len(parts)-1]
		ident = strings.Trim(ident, "`\"[]")
	}
	return strings.ToLower(ident)
}
