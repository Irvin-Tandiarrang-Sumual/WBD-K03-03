package sqlutil

import (
	"regexp"
	"strings"
)

var tableRename = regexp.MustCompile(`(?i)^(ALTER\s+TABLE|RENAME\s+TABLE)\s+(\S+)\s+(?:RENAME\s+)?TO\s+(\S+)$`)
var columnRename = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(\S+)\s+RENAME\s+COLUMN\s+(\S+)\s+TO\s+(\S+)$`)

func InverseRename(statement string) (string, bool) {
	if m := columnRename.FindStringSubmatch(statement); m != nil && IsValidIdentifier(m[1]) && IsValidIdentifier(m[2]) && IsValidIdentifier(m[3]) {
		return "ALTER TABLE " + m[1] + " RENAME COLUMN " + m[3] + " TO " + m[2], true
	}
	if m := tableRename.FindStringSubmatch(statement); m != nil && IsValidIdentifier(m[2]) && IsValidIdentifier(m[3]) {
		if strings.HasPrefix(strings.ToUpper(m[1]), "RENAME") {
			return "RENAME TABLE " + m[3] + " TO " + m[2], true
		}
		oldName := strings.Split(m[2], ".")
		newName := m[3]
		if len(oldName) > 1 && !strings.Contains(newName, ".") {
			newName = strings.Join(oldName[:len(oldName)-1], ".") + "." + newName
		}
		return "ALTER TABLE " + newName + " RENAME TO " + oldName[len(oldName)-1], true
	}
	return "", false
}
