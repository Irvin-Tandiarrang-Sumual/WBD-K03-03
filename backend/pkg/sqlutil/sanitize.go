package sqlutil

import (
	"errors"
	"strings"
)

var mutatingKeywords = []string{
	"delete", "update", "insert", "drop", "alter",
	"truncate", "create", "grant", "revoke", "merge",
	"call", "execute",
}

func IsReadOnlyQuery(query string) (bool, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return false, nil
	}

	sanitized, err := StripCommentsAndLiterals(trimmed)
	if err != nil {
		return false, err
	}

	if HasMultipleStatements(sanitized) {
		return false, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(sanitized))
	if !HasAllowedSelectPrefix(normalized) {
		return false, nil
	}

	if ContainsMutatingKeyword(normalized) {
		return false, nil
	}

	return true, nil
}

func HasAllowedSelectPrefix(normalizedQuery string) bool {
	return HasKeywordPrefix(normalizedQuery, "select") || HasKeywordPrefix(normalizedQuery, "with")
}

func HasMultipleStatements(sanitizedQuery string) bool {
	return strings.Contains(sanitizedQuery, ";")
}

func ContainsMutatingKeyword(normalizedQuery string) bool {
	for _, keyword := range mutatingKeywords {
		if ContainsWord(normalizedQuery, keyword) {
			return true
		}
	}
	return false
}

func StripCommentsAndLiterals(sql string) (string, error) {
	var builder strings.Builder
	builder.Grow(len(sql))

	index := 0
	length := len(sql)

	for index < length {
		if isLineCommentStart(sql, index) {
			index = consumeLineComment(sql, index)
			builder.WriteByte(' ')
			continue
		}

		if isBlockCommentStart(sql, index) {
			nextIndex, err := consumeBlockComment(sql, index)
			if err != nil {
				return "", err
			}
			index = nextIndex
			builder.WriteByte(' ')
			continue
		}

		if sql[index] == '\'' {
			nextIndex, err := consumeStringLiteral(sql, index)
			if err != nil {
				return "", err
			}
			index = nextIndex
			builder.WriteString(" '' ")
			continue
		}

		if sql[index] == '"' || sql[index] == '`' {
			quoteChar := sql[index]
			nextIndex, err := consumeQuotedIdentifier(sql, index, quoteChar)
			if err != nil {
				return "", err
			}
			index = nextIndex
			builder.WriteString(" ")
			continue
		}

		builder.WriteByte(sql[index])
		index++
	}

	return builder.String(), nil
}

func isLineCommentStart(sql string, index int) bool {
	return index+1 < len(sql) && sql[index] == '-' && sql[index+1] == '-'
}

func consumeLineComment(sql string, start int) int {
	index := start + 2
	for index < len(sql) && sql[index] != '\n' && sql[index] != '\r' {
		index++
	}
	return index
}

func isBlockCommentStart(sql string, index int) bool {
	return index+1 < len(sql) && sql[index] == '/' && sql[index+1] == '*'
}

func consumeBlockComment(sql string, start int) (int, error) {
	index := start + 2
	for index+1 < len(sql) {
		if sql[index] == '*' && sql[index+1] == '/' {
			return index + 2, nil
		}
		index++
	}
	return 0, errors.New("unclosed block comment")
}

func consumeStringLiteral(sql string, start int) (int, error) {
	index := start + 1
	length := len(sql)

	for index < length {
		if sql[index] == '\'' {
			if index+1 < length && sql[index+1] == '\'' {
				index += 2
				continue
			}
			return index + 1, nil
		}
		if sql[index] == '\\' && index+1 < length {
			index += 2
			continue
		}
		index++
	}
	return 0, errors.New("unclosed string literal")
}

func consumeQuotedIdentifier(sql string, start int, quoteChar byte) (int, error) {
	index := start + 1
	length := len(sql)

	for index < length {
		if sql[index] == quoteChar {
			if index+1 < length && sql[index+1] == quoteChar {
				index += 2
				continue
			}
			return index + 1, nil
		}
		index++
	}
	return 0, errors.New("unclosed quoted identifier")
}

func ContainsWord(text, word string) bool {
	searchOffset := 0
	for {
		relativeIndex := strings.Index(text[searchOffset:], word)
		if relativeIndex == -1 {
			return false
		}

		wordStart := searchOffset + relativeIndex
		wordEnd := wordStart + len(word)

		hasLeftBoundary := wordStart == 0 || !IsWordCharacter(text[wordStart-1])
		hasRightBoundary := wordEnd == len(text) || !IsWordCharacter(text[wordEnd])

		if hasLeftBoundary && hasRightBoundary {
			return true
		}

		searchOffset = wordStart + 1
	}
}

func IsWordCharacter(char byte) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_'
}

func HasKeywordPrefix(normalizedQuery, keyword string) bool {
	if !strings.HasPrefix(normalizedQuery, keyword) {
		return false
	}
	rest := normalizedQuery[len(keyword):]
	if rest == "" {
		return false
	}
	first := rest[0]
	return first == ' ' || first == '\n' || first == '\t' || first == '\r' || first == '*' || first == '('
}
