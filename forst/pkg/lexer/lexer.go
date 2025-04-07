package lexer

import (
	"bufio"
	"bytes"
	"unicode"

	"forst/pkg/ast"
)

// Lexer context for tracking file information
type Context struct {
	FilePath string
}

// Lexer: Converts Forst code into tokens
func Lexer(input []byte, ctx Context) []ast.Token {
	tokens := []ast.Token{}
	reader := bufio.NewReader(bytes.NewReader(input))
	path := ctx.FilePath
	lineNum := 0

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			break
		}
		lineNum++
		line = bytes.TrimRight(line, "\n")
		column := 0

		// Process each word in the line
		for len(line[column:]) > 0 {
			// Skip whitespace
			for column < len(line) && unicode.IsSpace(rune(line[column])) {
				column++
			}
			if column >= len(line) {
				break
			}

			// Check for string literals
			if line[column] == '"' {
				token, newColumn := processStringLiteral(line, column, path, lineNum)
				tokens = append(tokens, token)
				column = newColumn
				continue
			}

			// Check for special characters that should be separate tokens
			if isSpecialChar(line[column]) {
				token, newColumn := processSpecialChar(line, column, path, lineNum)
				tokens = append(tokens, token)
				column = newColumn

				if token.Type == ast.TokenComment {
					break
				}

				continue
			}

			// Process regular words
			token, newColumn := processWord(line, column, path, lineNum)
			tokens = append(tokens, token)
			column = newColumn
		}
	}

	// End of file token with final line/column position
	tokens = append(tokens, ast.Token{
		Type:   ast.TokenEOF,
		Value:  "",
		Path:   path,
		Line:   lineNum,
		Column: 1,
	})

	return tokens
}
