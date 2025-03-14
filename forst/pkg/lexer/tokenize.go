package lexer

import (
	"forst/pkg/ast"
	"unicode"
)

// Helper functions for token identification and processing
func isSpecialChar(c byte) bool {
	return c == '(' || c == ')' || c == '{' || c == '}' || c == ':' || c == ',' ||
		c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '=' ||
		c == '!' || c == '>' || c == '<' || c == '&' || c == '|'
}

func isTwoCharOperator(s string) bool {
	return s == "->" || s == "==" || s == "!=" || s == ">=" || s == "<=" || s == "&&" || s == "||"
}

func getTokenType(word string) ast.TokenType {
	switch word {
	case "fn":
		return ast.TokenFunc
	case "->":
		return ast.TokenArrow
	case "(":
		return ast.TokenLParen
	case ")":
		return ast.TokenRParen
	case "{":
		return ast.TokenLBrace
	case "}":
		return ast.TokenRBrace
	case "return":
		return ast.TokenReturn
	case "ensure":
		return ast.TokenEnsure
	case "or":
		return ast.TokenOr
	case "+":
		return ast.TokenPlus
	case "-":
		return ast.TokenMinus
	case "*":
		return ast.TokenMultiply
	case "/":
		return ast.TokenDivide
	case "%":
		return ast.TokenModulo
	case "==":
		return ast.TokenEquals
	case "!=":
		return ast.TokenNotEquals
	case ">":
		return ast.TokenGreater
	case "<":
		return ast.TokenLess
	case ">=":
		return ast.TokenGreaterEqual
	case "<=":
		return ast.TokenLessEqual
	case "&&":
		return ast.TokenAnd
	case "||":
		return ast.TokenOr
	case "!":
		return ast.TokenNot
	case ":":
		return ast.TokenColon
	case ",":
		return ast.TokenComma
	default:
		if unicode.IsDigit(rune(word[0])) {
			return ast.TokenInt
		}
		return ast.TokenIdent
	}
}

// Token processing functions
func processStringLiteral(line []byte, startCol int, path string, lineNum int) (ast.Token, int) {
	column := startCol
	column++ // Move past opening quote
	for column < len(line) && line[column] != '"' {
		column++
	}
	if column < len(line) {
		column++ // Move past closing quote
	}
	word := string(line[startCol:column])
	
	return ast.Token{
		Path:   path,
		Line:   lineNum,
		Column: startCol + 1,
		Value:  word,
		Type:   ast.TokenString,
	}, column
}

func processSpecialChar(line []byte, startCol int, path string, lineNum int) (ast.Token, int) {
	column := startCol
	column++
	// Handle two-character operators
	if column < len(line) && isTwoCharOperator(string(line[startCol:column+1])) {
		column++
	}
	word := string(line[startCol:column])

	token := ast.Token{
		Path:   path,
		Line:   lineNum,
		Column: startCol + 1,
		Value:  word,
		Type:   getTokenType(word),
	}
	
	return token, column
}

func processWord(line []byte, startCol int, path string, lineNum int) (ast.Token, int) {
	column := startCol
	for column < len(line) && !unicode.IsSpace(rune(line[column])) && !isSpecialChar(line[column]) {
		column++
	}
	word := string(line[startCol:column])

	return ast.Token{
		Path:   path,
		Line:   lineNum,
		Column: startCol + 1,
		Value:  word,
		Type:   getTokenType(word),
	}, column
}