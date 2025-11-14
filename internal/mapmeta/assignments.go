package mapmeta

import (
	"unicode"
)

type assignment struct {
	Name string
	Expr string
}

func scanAssignments(input string) []assignment {
	assignments := make([]assignment, 0, 64)
	length := len(input)

	for i := 0; i < length; {
		ch := input[i]

		if unicode.IsSpace(rune(ch)) {
			i++
			continue
		}

		if !isIdentifierStart(ch) {
			i++
			continue
		}

		start := i
		i++
		for i < length && isIdentifierPart(input[i]) {
			i++
		}
		name := input[start:i]

		j := i
		for j < length && unicode.IsSpace(rune(input[j])) && input[j] != '\n' {
			j++
		}
		if j >= length || input[j] != '=' {
			continue
		}

		// ensure we are not looking at '==' or other comparisons
		if j+1 < length {
			next := input[j+1]
			if next == '=' || next == '>' || next == '<' {
				i = j + 1
				continue
			}
		}

		// move past '=' and any additional whitespace
		j++
		for j < length && unicode.IsSpace(rune(input[j])) && input[j] != '\n' {
			j++
		}
		exprStart := j
		expr, nextPos := readExpression(input, exprStart)
		if expr != "" {
			assignments = append(assignments, assignment{
				Name: name,
				Expr: expr,
			})
		}
		i = nextPos
	}

	return assignments
}

func readExpression(input string, start int) (string, int) {
	length := len(input)
	depth := 0
	inString := false
	var quote byte
	escaped := false

	i := start
	for i < length {
		ch := input[i]

		if inString {
			if escaped {
				escaped = false
			} else {
				if ch == '\\' {
					escaped = true
				} else if ch == quote {
					inString = false
				}
			}
			i++
			continue
		}

		switch ch {
		case '\n':
			if depth == 0 {
				return trimWhitespace(input[start:i]), i
			}
		case '#':
			if depth == 0 {
				return trimWhitespace(input[start:i]), skipLine(input, i)
			}
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
			if depth == 0 && ch == ')' && i+1 < length && input[i+1] == '\n' {
				i++
				return trimWhitespace(input[start:i]), i
			}
		case '\'', '"':
			inString = true
			quote = ch
		}

		i++
	}
	return trimWhitespace(input[start:i]), length
}

func skipLine(input string, pos int) int {
	for pos < len(input) && input[pos] != '\n' {
		pos++
	}
	return pos
}

func trimWhitespace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func isIdentifierStart(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isIdentifierPart(b byte) bool {
	return isIdentifierStart(b) || (b >= '0' && b <= '9')
}
