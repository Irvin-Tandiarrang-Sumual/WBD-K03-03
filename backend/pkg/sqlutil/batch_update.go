package sqlutil

import (
	"regexp"
	"strings"
)

var batchUpdate = regexp.MustCompile(`(?is)^UPDATE\s+([a-z_][a-z_0-9]*(?:\.[a-z_][a-z_0-9]*)?)\s+SET\s+.+\s+WHERE\s+.+$`)
var unsafeBatchTerms = regexp.MustCompile(`(?i)\b(OR|NOT|RETURNING|SELECT|FROM)\b`)
var batchColumnName = regexp.MustCompile(`^[a-z_][a-z_0-9]*$`)

func BindBatchUpdate(script, column string) (string, string, error) {
	column = strings.ToLower(strings.TrimSpace(column))
	if !batchColumnName.MatchString(column) {
		return "", "", ErrInvalidSQLStatement
	}
	script = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(script), ";"))
	for strings.HasPrefix(script, "--") {
		_, rest, found := strings.Cut(script, "\n")
		if !found {
			return "", "", ErrInvalidSQLStatement
		}
		script = strings.TrimSpace(rest)
	}
	clean, err := StripCommentsAndLiterals(script)
	if err != nil || strings.Contains(clean, ";") || unsafeBatchTerms.MatchString(clean) ||
		strings.Count(script, "?") != 2 || strings.ContainsAny(script, "$`\\") || strings.Contains(script, "/*") || strings.Contains(script, "--") {
		return "", "", ErrInvalidSQLStatement
	}
	table := batchUpdate.FindStringSubmatch(clean)
	if len(table) != 2 {
		return "", "", ErrInvalidSQLStatement
	}
	changesKey := regexp.MustCompile(`(?i)\b` + column + `\s*=`)
	if changesKey.MatchString(clean) || strings.ContainsAny(script, "\"") {
		return "", "", ErrInvalidSQLStatement
	}
	bounds := regexp.MustCompile(`(?is)\b` + column + `\s+BETWEEN\s+\?\s+AND\s+\?$`)
	boundary := bounds.FindStringIndex(script)
	if boundary == nil {
		return "", "", ErrInvalidSQLStatement
	}
	return strings.ToLower(table[1]), script[:boundary[0]] + column + " IN ?", nil
}
