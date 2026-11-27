package domain

import (
	"regexp"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/sqlutil"
)

var destructiveSQL = regexp.MustCompile(`(?i)\b(DROP\s+(?:TABLE|DATABASE|SCHEMA|COLUMN)|TRUNCATE|DELETE\s+FROM)\b`)
var alterDrop = regexp.MustCompile(`(?is)\bALTER\s+TABLE\b[^;]*?\bDROP\b`)

func ClassifyMigration(sql string) (bool, string) {
	clean, err := sqlutil.StripCommentsAndLiterals(sql)
	if err != nil {
		return false, "Unable to classify SQL"
	}
	operation := destructiveSQL.FindString(clean)
	if operation == "" && alterDrop.MatchString(clean) {
		operation = "ALTER TABLE DROP"
	}
	if operation != "" {
		return false, strings.ToUpper(strings.Join(strings.Fields(operation), " ")) + " can permanently remove data"
	}
	return true, ""
}
