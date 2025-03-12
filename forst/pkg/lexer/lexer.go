package lexer

import (
	"strings"
	"unicode"

	"forst/pkg/ast"
)

// Lexer: Converts Forst code into tokens
func Lexer(input string) []ast.Token {
	tokens := []ast.Token{}
	words := strings.Fields(input)

	for _, word := range words {
		switch word {
		case "fn":
			tokens = append(tokens, ast.Token{Type: ast.TokenFunc, Value: word})
		case "->":
			tokens = append(tokens, ast.Token{Type: ast.TokenArrow, Value: word})
		case "(":
			tokens = append(tokens, ast.Token{Type: ast.TokenLParen, Value: word})
		case ")":
			tokens = append(tokens, ast.Token{Type: ast.TokenRParen, Value: word})
		case "{":
			tokens = append(tokens, ast.Token{Type: ast.TokenLBrace, Value: word})
		case "}":
			tokens = append(tokens, ast.Token{Type: ast.TokenRBrace, Value: word})
		case "return":
			tokens = append(tokens, ast.Token{Type: ast.TokenReturn, Value: word})
		case "assert":
			tokens = append(tokens, ast.Token{Type: ast.TokenAssert, Value: word})
		case "or":
			tokens = append(tokens, ast.Token{Type: ast.TokenOr, Value: word})
		case "+":
			tokens = append(tokens, ast.Token{Type: ast.TokenPlus, Value: word})
		case "-":
			tokens = append(tokens, ast.Token{Type: ast.TokenMinus, Value: word})
		case "*":
			tokens = append(tokens, ast.Token{Type: ast.TokenMultiply, Value: word})
		case "/":
			tokens = append(tokens, ast.Token{Type: ast.TokenDivide, Value: word})
		case "%":
			tokens = append(tokens, ast.Token{Type: ast.TokenModulo, Value: word})
		case "==":
			tokens = append(tokens, ast.Token{Type: ast.TokenEquals, Value: word})
		case "!=":
			tokens = append(tokens, ast.Token{Type: ast.TokenNotEquals, Value: word})
		case ">":
			tokens = append(tokens, ast.Token{Type: ast.TokenGreater, Value: word})
		case "<":
			tokens = append(tokens, ast.Token{Type: ast.TokenLess, Value: word})
		case ">=":
			tokens = append(tokens, ast.Token{Type: ast.TokenGreaterEqual, Value: word})
		case "<=":
			tokens = append(tokens, ast.Token{Type: ast.TokenLessEqual, Value: word})
		case "&&":
			tokens = append(tokens, ast.Token{Type: ast.TokenAnd, Value: word})
		case "||":
			tokens = append(tokens, ast.Token{Type: ast.TokenOr, Value: word})
		case "!":
			tokens = append(tokens, ast.Token{Type: ast.TokenNot, Value: word})
		default:
			if unicode.IsDigit(rune(word[0])) {
				tokens = append(tokens, ast.Token{Type: ast.TokenInt, Value: word})
			} else {
				tokens = append(tokens, ast.Token{Type: ast.TokenIdent, Value: word})
			}
		}
	}

	tokens = append(tokens, ast.Token{Type: ast.TokenEOF, Value: ""}) // End of file token
	return tokens
}