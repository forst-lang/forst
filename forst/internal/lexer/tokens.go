package lexer

import (
	"forst/internal/ast"
)

// Keywords maps Forst keywords to their token types for static analysis
var Keywords = map[string]ast.TokenIdent{
	"func":        ast.TokenFunc,
	"import":      ast.TokenImport,
	"package":     ast.TokenPackage,
	"String":      ast.TokenString,
	"Int":         ast.TokenInt,
	"Float":       ast.TokenFloat,
	"Bool":        ast.TokenBool,
	"Void":        ast.TokenVoid,
	"Array":       ast.TokenArray,
	"return":      ast.TokenReturn,
	"ensure":      ast.TokenEnsure,
	"is":          ast.TokenIs,
	"or":          ast.TokenOr,
	"type":        ast.TokenType,
	"if":          ast.TokenIf,
	"else if":     ast.TokenElseIf,
	"else":        ast.TokenElse,
	"for":         ast.TokenFor,
	"range":       ast.TokenRange,
	"break":       ast.TokenBreak,
	"continue":    ast.TokenContinue,
	"switch":      ast.TokenSwitch,
	"case":        ast.TokenCase,
	"default":     ast.TokenDefault,
	"fallthrough": ast.TokenFallthrough,
	"var":         ast.TokenVar,
	"const":       ast.TokenConst,
	"map":         ast.TokenMap,
	"chan":        ast.TokenChan,
	"interface":   ast.TokenInterface,
	"struct":      ast.TokenStruct,
	"go":          ast.TokenGo,
	"defer":       ast.TokenDefer,
	"goto":        ast.TokenGoto,
	"nil":         ast.TokenNil,
}

// Operators maps Forst operators to their token types
var Operators = map[string]ast.TokenIdent{
	"//": ast.TokenComment,
	"(":  ast.TokenLParen,
	")":  ast.TokenRParen,
	"{":  ast.TokenLBrace,
	"}":  ast.TokenRBrace,
	"[":  ast.TokenLBracket,
	"]":  ast.TokenRBracket,
	"+":  ast.TokenPlus,
	"-":  ast.TokenMinus,
	"*":  ast.TokenStar,
	"/":  ast.TokenDivide,
	"%":  ast.TokenModulo,
	"==": ast.TokenEquals,
	"!=": ast.TokenNotEquals,
	">":  ast.TokenGreater,
	"<":  ast.TokenLess,
	">=": ast.TokenGreaterEqual,
	"<=": ast.TokenLessEqual,
	"&&": ast.TokenLogicalAnd,
	"||": ast.TokenLogicalOr,
	":=": ast.TokenColonEquals,
	"!":  ast.TokenLogicalNot,
	":":  ast.TokenColon,
	",":  ast.TokenComma,
	".":  ast.TokenDot,
	";":  ast.TokenSemicolon,
	"=":  ast.TokenEquals,
	"&":  ast.TokenBitwiseAnd,
	"|":  ast.TokenBitwiseOr,
	"->": ast.TokenArrow,
}

// GetTokenType returns the token type for a given word
func GetTokenType(word string) ast.TokenIdent {
	// Check keywords first
	if tokenType, ok := Keywords[word]; ok {
		return tokenType
	}

	// Check operators
	if tokenType, ok := Operators[word]; ok {
		return tokenType
	}

	// Check for numeric literals
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
