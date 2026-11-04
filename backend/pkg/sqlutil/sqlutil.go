package sqlutil

import (
	"fmt"
	"strings"
)

type sqlScanner struct {
	script         string
	index          int
	inSingleQuote  bool
	inDoubleQuote  bool
	inBacktick     bool
	inLineComment  bool
	inBlockComment bool
}

func newSQLScanner(script string) *sqlScanner {
	return &sqlScanner{script: script}
}

func (s *sqlScanner) hasMore() bool {
	return s.index < len(s.script)
}

func (s *sqlScanner) current() byte {
	return s.script[s.index]
}

func (s *sqlScanner) peek() (byte, bool) {
	if s.index+1 < len(s.script) {
		return s.script[s.index+1], true
	}
	return 0, false
}

func (s *sqlScanner) advance() {
	s.index++
}

func (s *sqlScanner) isInQuotes() bool {
	return s.inSingleQuote || s.inDoubleQuote || s.inBacktick
}

func (s *sqlScanner) isInComment() bool {
	return s.inLineComment || s.inBlockComment
}

func (s *sqlScanner) handleInsideComment(buffer *strings.Builder) bool {
	if !s.isInComment() {
		return false
	}

	ch := s.current()
	if s.inLineComment && ch == '\n' {
		s.inLineComment = false
		buffer.WriteByte(ch)
	} else if s.inBlockComment && ch == '*' {
		if next, ok := s.peek(); ok && next == '/' {
			s.inBlockComment = false
			s.advance()
		}
	}

	s.advance()
	return true
}

func (s *sqlScanner) handleCommentStart() bool {
	if s.isInQuotes() {
		return false
	}

	ch := s.current()
	next, hasNext := s.peek()
	if !hasNext {
		return false
	}

	if ch == '-' && next == '-' {
		s.inLineComment = true
		s.advance()
		s.advance()
		return true
	}

	if ch == '/' && next == '*' {
		s.inBlockComment = true
		s.advance()
		s.advance()
		return true
	}

	return false
}

func (s *sqlScanner) handleQuotes(buffer *strings.Builder) bool {
	ch := s.current()

	if ch == '\'' && !s.inDoubleQuote && !s.inBacktick {
		buffer.WriteByte(ch)
		if s.inSingleQuote {
			if next, ok := s.peek(); ok && next == '\'' {
				s.advance()
				buffer.WriteByte(s.current())
				s.advance()
				return true
			}
		}
		s.inSingleQuote = !s.inSingleQuote
		s.advance()
		return true
	}

	if ch == '"' && !s.inSingleQuote && !s.inBacktick {
		s.inDoubleQuote = !s.inDoubleQuote
		buffer.WriteByte(ch)
		s.advance()
		return true
	}

	if ch == '`' && !s.inSingleQuote && !s.inDoubleQuote {
		s.inBacktick = !s.inBacktick
		buffer.WriteByte(ch)
		s.advance()
		return true
	}

	return false
}

func (s *sqlScanner) isStatementDelimiter() bool {
	return s.current() == ';' && !s.isInQuotes()
}

func SplitStatements(sqlScript string) []string {
	statements := make([]string, 0)
	var current strings.Builder
	scanner := newSQLScanner(sqlScript)

	for scanner.hasMore() {
		if scanner.handleInsideComment(&current) {
			continue
		}

		if scanner.handleCommentStart() {
			continue
		}

		if scanner.handleQuotes(&current) {
			continue
		}

		if scanner.isStatementDelimiter() {
			if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
				statements = append(statements, trimmed)
			}
			current.Reset()
			scanner.advance()
			continue
		}

		current.WriteByte(scanner.current())
		scanner.advance()
	}

	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		statements = append(statements, trimmed)
	}

	return statements
}

func SplitValues(input string) []string {
	values := make([]string, 0)
	start := 0
	depth := 0
	quoted := false

	for i, ch := range input {
		switch ch {
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
				values = append(values, strings.TrimSpace(input[start:i]))
				start = i + 1
			}
		}
	}

	values = append(values, strings.TrimSpace(input[start:]))
	return values
}

func SplitValueGroups(input string) ([]string, bool) {
	groups := make([]string, 0)
	depth := 0
	start := -1
	quoted := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if isEscapedQuote(input, i, quoted) {
			i++
			continue
		}

		if ch == '\'' {
			quoted = !quoted
			continue
		}

		if quoted {
			continue
		}

		switch ch {
		case '(':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')':
			depth--
			if depth < 0 || (depth == 0 && start < 0) {
				return nil, false
			}
			if depth == 0 {
				groups = append(groups, strings.TrimSpace(input[start:i]))
				start = -1
			}
		default:
			if isInvalidCharacterOutsideGroup(ch, depth) {
				return nil, false
			}
		}
	}

	return groups, isBalancedGroup(quoted, depth, start)
}

func isEscapedQuote(input string, index int, quoted bool) bool {
	return input[index] == '\'' && quoted && index+1 < len(input) && input[index+1] == '\''
}

func isInvalidCharacterOutsideGroup(ch byte, depth int) bool {
	return depth == 0 && ch != ',' && !IsWhitespace(ch)
}

func isBalancedGroup(quoted bool, depth int, start int) bool {
	return !quoted && depth == 0 && start == -1
}

func ParseAssignments(input string) map[string]string {
	assignments := make(map[string]string)
	for _, assignment := range SplitValues(input) {
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) == 2 {
			assignments[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return assignments
}

func FormatSQLValue(value any) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case []byte:
		return "'" + strings.ReplaceAll(string(v), "'", "''") + "'"
	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprint(v)
	}
}

func IsWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func ReverseStrings(items []string) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
