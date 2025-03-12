package lexer

import (
	"bufio"
	"strings"
	"unicode"

	"forst/pkg/ast"
)

// Lexer: Converts Forst code into tokens
func Lexer(input string) []ast.Token {
	tokens := []ast.Token{}
	reader := bufio.NewReader(strings.NewReader(input))
	filename := "input.forst" // Default filename
	lineNum := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			break
		}
		lineNum++
		line = strings.TrimRight(line, "\n")
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
				wordStart := column
				column++ // Move past opening quote
				for column < len(line) && line[column] != '"' {
					column++
				}
				if column < len(line) {
					column++ // Move past closing quote
				}
				word := line[wordStart:column]
				token := ast.Token{
					Filename: filename,
					Line:     lineNum,
					Column:   wordStart + 1,
					Value:    word,
					Type:     ast.TokenString,
				}
				tokens = append(tokens, token)
				continue
			}

			// Find next word
			wordStart := column
			for column < len(line) && !unicode.IsSpace(rune(line[column])) {
				column++
			}
			word := line[wordStart:column]

			token := ast.Token{
				Filename: filename,
				Line:     lineNum,
				Column:   wordStart + 1,
				Value:    word,
			}

			switch word {
			case "fn":
				token.Type = ast.TokenFunc
			case "->":
				token.Type = ast.TokenArrow
			case "(":
				token.Type = ast.TokenLParen
			case ")":
				token.Type = ast.TokenRParen
			case "{":
				token.Type = ast.TokenLBrace
			case "}":
				token.Type = ast.TokenRBrace
			case "return":
				token.Type = ast.TokenReturn
			case "assert":
				token.Type = ast.TokenAssert
			case "or":
				token.Type = ast.TokenOr
			case "+":
				token.Type = ast.TokenPlus
			case "-":
				token.Type = ast.TokenMinus
			case "*":
				token.Type = ast.TokenMultiply
			case "/":
				token.Type = ast.TokenDivide
			case "%":
				token.Type = ast.TokenModulo
			case "==":
				token.Type = ast.TokenEquals
			case "!=":
				token.Type = ast.TokenNotEquals
			case ">":
				token.Type = ast.TokenGreater
			case "<":
				token.Type = ast.TokenLess
			case ">=":
				token.Type = ast.TokenGreaterEqual
			case "<=":
				token.Type = ast.TokenLessEqual
			case "&&":
				token.Type = ast.TokenAnd
			case "||":
				token.Type = ast.TokenOr
			case "!":
				token.Type = ast.TokenNot
			default:
				if unicode.IsDigit(rune(word[0])) {
					token.Type = ast.TokenInt
				} else {
					token.Type = ast.TokenIdent
				}
			}

			tokens = append(tokens, token)
		}
	}

	// End of file token with final line/column position
	lastCol := 1
	tokens = append(tokens, ast.Token{
		Type:     ast.TokenEOF,
		Value:    "",
		Filename: filename,
		Line:     lineNum,
		Column:   lastCol,
	})

	return tokens
}