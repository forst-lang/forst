package ast

type TokenType string

const (
	// Namespacing
	TokenImport  TokenType = "IMPORT"
	TokenPackage TokenType = "PACKAGE"

	// User-defined identifiers
	TokenIdentifier TokenType = "IDENTIFIER"

	// Language keywords
	TokenFunction TokenType = "FUNCTION"

	// Built-in types
	TokenInt    TokenType = "TYPE_INT"
	TokenFloat  TokenType = "TYPE_FLOAT"
	TokenString TokenType = "TYPE_STRING"
	TokenBool   TokenType = "TYPE_BOOL"
	TokenArray  TokenType = "TYPE_ARRAY"
	TokenStruct TokenType = "TYPE_STRUCT"

	// Literals
	TokenIntLiteral    TokenType = "INT_LITERAL"
	TokenFloatLiteral  TokenType = "FLOAT_LITERAL"
	TokenStringLiteral TokenType = "STRING_LITERAL"
	TokenTrue          TokenType = "TRUE_LITERAL"
	TokenFalse         TokenType = "FALSE_LITERAL"

	// Punctuation
	TokenLParen      TokenType = "LPAREN"
	TokenRParen      TokenType = "RPAREN"
	TokenLBrace      TokenType = "LBRACE"
	TokenRBrace      TokenType = "RBRACE"
	TokenReturn      TokenType = "RETURN"
	TokenEnsure      TokenType = "ENSURE"
	TokenIs          TokenType = "IS"
	TokenOr          TokenType = "OR"
	TokenEOF         TokenType = "EOF"
	TokenColon       TokenType = "COLON"
	TokenComma       TokenType = "COMMA"
	TokenDot         TokenType = "DOT"
	TokenColonEquals TokenType = "COLON_EQUALS"

	// Arithmetic operators
	TokenPlus     TokenType = "PLUS"     // +
	TokenMinus    TokenType = "MINUS"    // -
	TokenMultiply TokenType = "MULTIPLY" // *
	TokenDivide   TokenType = "DIVIDE"   // /
	TokenModulo   TokenType = "MODULO"   // %

	// Comparison operators
	TokenEquals       TokenType = "EQUALS"        // ==
	TokenNotEquals    TokenType = "NOT_EQUALS"    // !=
	TokenGreater      TokenType = "GREATER"       // >
	TokenLess         TokenType = "LESS"          // <
	TokenGreaterEqual TokenType = "GREATER_EQUAL" // >=
	TokenLessEqual    TokenType = "LESS_EQUAL"    // <=

	// Logical operators
	TokenLogicalAnd TokenType = "LOGICAL_AND" // &&
	TokenLogicalOr  TokenType = "LOGICAL_OR"  // ||
	TokenLogicalNot TokenType = "LOGICAL_NOT" // !
)

// Token structure
type Token struct {
	Type   TokenType
	Value  string
	Path   string
	Line   int
	Column int
}

func (t TokenType) IsBinaryOperator() bool {
	return t == TokenPlus || t == TokenMinus ||
		t == TokenMultiply || t == TokenDivide ||
		t == TokenModulo || t == TokenEquals ||
		t == TokenNotEquals || t == TokenGreater ||
		t == TokenLess || t == TokenGreaterEqual ||
		t == TokenLessEqual || t == TokenLogicalAnd ||
		t == TokenLogicalOr
}

func (t TokenType) IsUnaryOperator() bool {
	return t == TokenLogicalNot
}

func (t TokenType) IsLiteral() bool {
	return t == TokenIntLiteral || t == TokenFloatLiteral || t == TokenStringLiteral || t == TokenTrue || t == TokenFalse
}

func (t TokenType) IsArithmeticBinaryOperator() bool {
	return t == TokenPlus || t == TokenMinus || t == TokenMultiply || t == TokenDivide || t == TokenModulo
}

func (t TokenType) IsComparisonBinaryOperator() bool {
	return t == TokenEquals || t == TokenNotEquals || t == TokenGreater || t == TokenLess || t == TokenGreaterEqual || t == TokenLessEqual
}

func (t TokenType) IsLogicalBinaryOperator() bool {
	return t == TokenLogicalAnd || t == TokenLogicalOr
}
