package lexer

import (
	"forst/internal/ast"
)

// GetTokenType returns the token type for a given word
func GetTokenType(word string) ast.TokenIdent {
	switch word {
	case "//":
		return ast.TokenComment
	case "func":
		return ast.TokenFunc
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
	case "Void":
		return ast.TokenVoid
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
	case "type":
		return ast.TokenType
	case "(":
		return ast.TokenLParen
	case ")":
		return ast.TokenRParen
	case "{":
		return ast.TokenLBrace
	case "}":
		return ast.TokenRBrace
	case "[":
		return ast.TokenLBracket
	case "]":
		return ast.TokenRBracket
	case "+":
		return ast.TokenPlus
	case "-":
		return ast.TokenMinus
	case "*":
		return ast.TokenStar
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
	case ";":
		return ast.TokenSemicolon
	case "=":
		return ast.TokenEquals
	case "&":
		return ast.TokenBitwiseAnd
	case "|":
		return ast.TokenBitwiseOr
	case "if":
		return ast.TokenIf
	case "else if":
		return ast.TokenElseIf
	case "else":
		return ast.TokenElse
	case "for":
		return ast.TokenFor
	case "range":
		return ast.TokenRange
	case "break":
		return ast.TokenBreak
	case "continue":
		return ast.TokenContinue
	case "switch":
		return ast.TokenSwitch
	case "case":
		return ast.TokenCase
	case "default":
		return ast.TokenDefault
	case "fallthrough":
		return ast.TokenFallthrough
	case "var":
		return ast.TokenVar
	case "const":
		return ast.TokenConst
	case "map":
		return ast.TokenMap
	case "chan":
		return ast.TokenChan
	case "->":
		return ast.TokenArrow
	case "interface":
		return ast.TokenInterface
	case "struct":
		return ast.TokenStruct
	case "go":
		return ast.TokenGo
	case "defer":
		return ast.TokenDefer
	case "goto":
		return ast.TokenGoto
	case "nil":
		return ast.TokenNil
	default:
		if isDigit(word[0]) {
			// Check for float literals
			for i := 0; i < len(word); i++ {
				if word[i] == '.' || word[i] == 'e' || word[i] == 'E' || word[i] == 'i' {
					return ast.TokenFloatLiteral
				}
			}
			return ast.TokenIntLiteral
		}
		return ast.TokenIdentifier
	}
}
