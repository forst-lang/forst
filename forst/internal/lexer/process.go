package lexer

import (
	"forst/internal/ast"
	"unicode"
)

// Token processing functions
func processStringLiteral(line []byte, startCol int, path string, lineNum int) (ast.Token, int) {
	column := startCol
	quoteChar := line[column]
	column++ // Move past opening quote

	// For backtick strings, we don't need to handle escapes
	if quoteChar == '`' {
		for column < len(line) && line[column] != '`' {
			column++
		}
		if column < len(line) {
			column++ // Move past closing backtick
		}
		word := string(line[startCol:column])
		return ast.Token{
			Path:   path,
			Line:   lineNum,
			Column: startCol + 1,
			Value:  word,
			Type:   ast.TokenStringLiteral,
		}, column
	}

	// For double and single quoted strings, handle escapes
	for column < len(line) {
		if line[column] == '\\' {
			column += 2 // Skip the escape sequence
			continue
		}
		if line[column] == quoteChar {
			column++ // Move past closing quote
			break
		}
		column++
	}

	word := string(line[startCol:column])

	return ast.Token{
		Path:   path,
		Line:   lineNum,
		Column: startCol + 1,
		Value:  word,
		Type:   ast.TokenStringLiteral,
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

	// Special handling for line comments
	if word == "//" {
		commentContent := "//"
		for column < len(line) {
			commentContent += string(line[column])
			column++
		}
		return ast.Token{
			Path:   path,
			Line:   lineNum,
			Column: startCol + 1,
			Value:  commentContent,
			Type:   ast.TokenComment,
		}, column
	}

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
	// Special handling for numbers
	if isDigit(line[column]) {
		hasDecimal := false
		hasExponent := false
		hasImaginary := false

		// Check for base prefix
		if line[column] == '0' && column+1 < len(line) {
			if line[column+1] == 'x' || line[column+1] == 'X' {
				column += 2
			} else if line[column+1] == 'b' || line[column+1] == 'B' {
				column += 2
			} else if line[column+1] == 'o' || line[column+1] == 'O' {
				column += 2
			}
		}

		// Parse digits
		for column < len(line) {
			c := line[column]
			if c == '.' && !hasDecimal && !hasExponent {
				hasDecimal = true
				column++
				continue
			}
			if (c == 'e' || c == 'E') && !hasExponent {
				hasExponent = true
				column++
				if column < len(line) && (line[column] == '+' || line[column] == '-') {
					column++
				}
				continue
			}
			if c == 'i' && !hasImaginary {
				hasImaginary = true
				column++
				break
			}
			if !isDigit(c) && !unicode.IsLetter(rune(c)) {
				break
			}
			column++
		}

		word := string(line[startCol:column])
		tokenType := ast.TokenIntLiteral
		if hasDecimal || hasExponent || hasImaginary {
			tokenType = ast.TokenFloatLiteral
		}

		return ast.Token{
			Path:   path,
			Line:   lineNum,
			Column: startCol + 1,
			Value:  word,
			Type:   tokenType,
		}, column
	}

	// Regular word processing
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
