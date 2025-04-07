package ast

type TokenIdent string

const (
	// Namespacing
	TokenImport  TokenIdent = "IMPORT"
	TokenPackage TokenIdent = "PACKAGE"

	// User-defined identifiers
	TokenIdentifier TokenIdent = "IDENTIFIER"

	// Language keywords
	TokenFunction TokenIdent = "FUNCTION"
	TokenType     TokenIdent = "TYPE"

	// Built-in types
	TokenInt    TokenIdent = "TYPE_INT"
	TokenFloat  TokenIdent = "TYPE_FLOAT"
	TokenString TokenIdent = "TYPE_STRING"
	TokenBool   TokenIdent = "TYPE_BOOL"
	TokenArray  TokenIdent = "TYPE_ARRAY"
	TokenStruct TokenIdent = "TYPE_STRUCT"
	TokenVoid   TokenIdent = "TYPE_VOID"

	// Literals
	TokenIntLiteral    TokenIdent = "INT_LITERAL"
	TokenFloatLiteral  TokenIdent = "FLOAT_LITERAL"
	TokenStringLiteral TokenIdent = "STRING_LITERAL"
	TokenTrue          TokenIdent = "TRUE_LITERAL"
	TokenFalse         TokenIdent = "FALSE_LITERAL"

	// Punctuation
	TokenLParen      TokenIdent = "LPAREN"
	TokenRParen      TokenIdent = "RPAREN"
	TokenLBrace      TokenIdent = "LBRACE"
	TokenRBrace      TokenIdent = "RBRACE"
	TokenReturn      TokenIdent = "RETURN"
	TokenEnsure      TokenIdent = "ENSURE"
	TokenIs          TokenIdent = "IS"
	TokenOr          TokenIdent = "OR"
	TokenEOF         TokenIdent = "EOF"
	TokenColon       TokenIdent = "COLON"
	TokenComma       TokenIdent = "COMMA"
	TokenDot         TokenIdent = "DOT"
	TokenColonEquals TokenIdent = "COLON_EQUALS"

	// Arithmetic operators
	TokenPlus     TokenIdent = "PLUS"     // +
	TokenMinus    TokenIdent = "MINUS"    // -
	TokenMultiply TokenIdent = "MULTIPLY" // *
	TokenDivide   TokenIdent = "DIVIDE"   // /
	TokenModulo   TokenIdent = "MODULO"   // %

	// Comparison operators
	TokenEquals       TokenIdent = "EQUALS"        // ==
	TokenNotEquals    TokenIdent = "NOT_EQUALS"    // !=
	TokenGreater      TokenIdent = "GREATER"       // >
	TokenLess         TokenIdent = "LESS"          // <
	TokenGreaterEqual TokenIdent = "GREATER_EQUAL" // >=
	TokenLessEqual    TokenIdent = "LESS_EQUAL"    // <=

	// Logical operators
	TokenLogicalAnd TokenIdent = "LOGICAL_AND" // &&
	TokenLogicalOr  TokenIdent = "LOGICAL_OR"  // ||
	TokenLogicalNot TokenIdent = "LOGICAL_NOT" // !

	// Bitwise operators
	TokenBitwiseAnd TokenIdent = "BITWISE_AND" // &
	TokenBitwiseOr  TokenIdent = "BITWISE_OR"  // |

	TokenComment TokenIdent = "COMMENT" // //
)

// Token structure
type Token struct {
	Type   TokenIdent
	Value  string
	Path   string
	Line   int
	Column int
}

func (t TokenIdent) IsBinaryOperator() bool {
	return t == TokenPlus || t == TokenMinus ||
		t == TokenMultiply || t == TokenDivide ||
		t == TokenModulo || t == TokenEquals ||
		t == TokenNotEquals || t == TokenGreater ||
		t == TokenLess || t == TokenGreaterEqual ||
		t == TokenLessEqual || t == TokenLogicalAnd ||
		t == TokenLogicalOr
}

func (t TokenIdent) IsUnaryOperator() bool {
	return t == TokenLogicalNot
}

func (t TokenIdent) IsLiteral() bool {
	return t == TokenIntLiteral || t == TokenFloatLiteral || t == TokenStringLiteral || t == TokenTrue || t == TokenFalse
}

func (t TokenIdent) IsArithmeticBinaryOperator() bool {
	return t == TokenPlus || t == TokenMinus || t == TokenMultiply || t == TokenDivide || t == TokenModulo
}

func (t TokenIdent) IsComparisonBinaryOperator() bool {
	return t == TokenEquals || t == TokenNotEquals || t == TokenGreater || t == TokenLess || t == TokenGreaterEqual || t == TokenLessEqual
}

func (t TokenIdent) IsLogicalBinaryOperator() bool {
	return t == TokenLogicalAnd || t == TokenLogicalOr
}
