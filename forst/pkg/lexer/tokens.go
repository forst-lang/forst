package lexer

import (
	"forst/pkg/ast"
)

func GetTokenType(word string) ast.TokenType {
	switch word {
	case "fn":
		return ast.TokenFunction
	case "import":
		return ast.TokenImport
	case "package":
		return ast.TokenPackage
	case "String":
		return ast.TokenString
	case "Int":
		return ast.TokenInt
	case "Float":
		return ast.TokenFloat
	case "Bool":
		return ast.TokenBool
	case "Array":
		return ast.TokenArray
	case "return":
		return ast.TokenReturn
	case "ensure":
		return ast.TokenEnsure
	case "is":
		return ast.TokenIs
	case "or":
		return ast.TokenOr
	case "(":
		return ast.TokenLParen
	case ")":
		return ast.TokenRParen
	case "{":
		return ast.TokenLBrace
	case "}":
		return ast.TokenRBrace
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
		return ast.TokenLogicalAnd
	case "||":
		return ast.TokenLogicalOr
	case ":=":
		return ast.TokenColonEquals
	case "!":
		return ast.TokenLogicalNot
	case ":":
		return ast.TokenColon
	case ",":
		return ast.TokenComma
	case ".":
		return ast.TokenDot
	case "=":
		return ast.TokenEquals
	default:
		if isDigit(word[0]) {
			lastChar := word[len(word)-1]
			if lastChar == 'f' {
				return ast.TokenFloatLiteral
			}
			return ast.TokenIntLiteral
		}
		return ast.TokenIdentifier
	}
}
