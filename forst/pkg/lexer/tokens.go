package lexer

import (
	"forst/pkg/ast"
	"unicode"
)

func GetTokenType(word string) ast.TokenType {
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
