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
	TokenLParen TokenType = "LPAREN"
	TokenRParen TokenType = "RPAREN"
	TokenLBrace TokenType = "LBRACE"
	TokenRBrace TokenType = "RBRACE"
	TokenReturn TokenType = "RETURN"
	TokenEnsure TokenType = "ENSURE"
	TokenOr     TokenType = "OR"
	TokenEOF    TokenType = "EOF"
	TokenColon  TokenType = "COLON"
	TokenComma  TokenType = "COMMA"

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
