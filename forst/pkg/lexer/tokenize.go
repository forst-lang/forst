package lexer

import (
	"forst/pkg/ast"
	"unicode"
)

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
		Type:   GetTokenType(word),
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
		Type:   GetTokenType(word),
	}, column
}