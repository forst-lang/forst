package lexer

import (
	"bufio"
	"bytes"
	"unicode"

	"forst/pkg/ast"
)

// Lexer: Converts Forst code into tokens
type Context struct {
	FilePath string
}

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
				wordStart := column
				column++ // Move past opening quote
				for column < len(line) && line[column] != '"' {
					column++
				}
				if column < len(line) {
					column++ // Move past closing quote
				}
				word := string(line[wordStart:column])
				token := ast.Token{
					Path:   path,
					Line:   lineNum,
					Column: wordStart + 1,
					Value:  word,
					Type:   ast.TokenString,
				}
				tokens = append(tokens, token)
				continue
			}

			// Check for special characters that should be separate tokens
			if isSpecialChar(line[column]) {
				wordStart := column
				column++
				// Handle two-character operators
				if column < len(line) && isTwoCharOperator(string(line[wordStart:column+1])) {
					column++
				}
				word := string(line[wordStart:column])

				token := ast.Token{
					Path:   path,
					Line:   lineNum,
					Column: wordStart + 1,
					Value:  word,
				}

				token.Type = getTokenType(word)
				tokens = append(tokens, token)
				continue
			}

			// Find next word
			wordStart := column
			for column < len(line) && !unicode.IsSpace(rune(line[column])) && !isSpecialChar(line[column]) {
				column++
			}
			word := string(line[wordStart:column])

			token := ast.Token{
				Path:   path,
				Line:   lineNum,
				Column: wordStart + 1,
				Value:  word,
			}

			token.Type = getTokenType(word)
			tokens = append(tokens, token)
		}
	}

	// End of file token with final line/column position
	lastCol := 1
	tokens = append(tokens, ast.Token{
		Type:   ast.TokenEOF,
		Value:  "",
		Path:   path,
		Line:   lineNum,
		Column: lastCol,
	})

	return tokens
}

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