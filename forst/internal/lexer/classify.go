package lexer

import (
	"unicode"
)

// Character classification utilities

// isSpecialChar checks if a character should become a separate token
func isSpecialChar(c byte) bool {
	return c == '(' || c == ')' || c == '{' || c == '}' || c == ':' || c == ',' ||
		c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '=' ||
		c == '!' || c == '>' || c == '<' || c == '&' || c == '|' || c == '.' ||
		c == '[' || c == ']' || c == ';' || c == '^'
}

// operatorSpan returns the byte length of the operator starting at line[start].
func operatorSpan(line []byte, start int) int {
	if start >= len(line) {
		return 0
	}
	if start+2 < len(line) {
		switch string(line[start : start+3]) {
		case "<<=", ">>=", "&^=":
			return 3
		}
	}
	if start+1 < len(line) {
		switch string(line[start : start+2]) {
		case "->", "<-", "==", "!=", ">=", "<=", "&&", "||", ":=", "//",
			"++", "--", "+=", "-=", "*=", "/=", "%=", "&=", "|=",
			"<<", ">>", "&^", "^=":
			return 2
		}
	}
	return 1
}

// isTwoCharOperator checks if a string is a two-character operator
func isTwoCharOperator(s string) bool {
	return len(s) == 2 && operatorSpan([]byte(s), 0) == 2
}

// isEllipsis checks for Go-style variadic spread token.
func isEllipsis(line []byte, start int) bool {
	return start+2 < len(line) && line[start] == '.' && line[start+1] == '.' && line[start+2] == '.'
}

// isDigit checks if a character is a digit
func isDigit(c byte) bool {
	return unicode.IsDigit(rune(c))
}
