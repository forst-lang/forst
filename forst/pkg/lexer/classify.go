package lexer

import (
	"unicode"
)

// Character classification utilities

// isSpecialChar checks if a character is a special token character
func isSpecialChar(c byte) bool {
	return c == '(' || c == ')' || c == '{' || c == '}' || c == ':' || c == ',' ||
		c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '=' ||
		c == '!' || c == '>' || c == '<' || c == '&' || c == '|' || c == '.'
}

// isTwoCharOperator checks if a string is a two-character operator
func isTwoCharOperator(s string) bool {
	return s == "->" || s == "==" || s == "!=" || s == ">=" || s == "<=" || s == "&&" || s == "||" || s == ":=" || s == "//"
}

// isDigit checks if a character is a digit
func isDigit(c byte) bool {
	return unicode.IsDigit(rune(c))
}
