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
		c == '[' || c == ']' || c == ';'
}

// isTwoCharOperator checks if a string is a two-character operator
func isTwoCharOperator(s string) bool {
	return s == "->" || s == "==" || s == "!=" || s == ">=" || s == "<=" || s == "&&" || s == "||" || s == ":=" || s == "//" ||
		s == "++" || s == "--" ||
		s == "+=" || s == "-=" || s == "*=" || s == "/=" || s == "%=" || s == "&=" || s == "|="
}

// isEllipsis checks for Go-style variadic spread token.
func isEllipsis(line []byte, start int) bool {
	return start+2 < len(line) && line[start] == '.' && line[start+1] == '.' && line[start+2] == '.'
}

// isDigit checks if a character is a digit
func isDigit(c byte) bool {
	return unicode.IsDigit(rune(c))
}
