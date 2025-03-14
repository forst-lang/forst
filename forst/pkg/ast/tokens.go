package ast

type TokenType string

const (
	// Namespacing
	TokenImport    TokenType = "IMPORT"
	TokenPackage   TokenType = "PACKAGE"

	// User-defined identifiers
	TokenIdent     TokenType = "IDENT"

	// Language keywords
	TokenFunc      TokenType = "FUNC"

	// Built-in types
	TokenInt       TokenType = "INT"
	TokenFloat     TokenType = "FLOAT"
	TokenString    TokenType = "STRING"
	TokenBool      TokenType = "BOOL"
	TokenArray     TokenType = "ARRAY"
	TokenStruct    TokenType = "STRUCT"

	// Punctuation
	TokenLParen    TokenType = "LPAREN"
	TokenRParen    TokenType = "RPAREN"
	TokenLBrace    TokenType = "LBRACE"
	TokenRBrace    TokenType = "RBRACE"
	TokenReturn    TokenType = "RETURN"
	TokenEnsure    TokenType = "ENSURE"
	TokenEOF       TokenType = "EOF"
	TokenColon     TokenType = "COLON"
	TokenComma     TokenType = "COMMA"

	// Arithmetic operators
	TokenPlus     TokenType = "PLUS"      // +
	TokenMinus    TokenType = "MINUS"     // -
	TokenMultiply TokenType = "MULTIPLY"  // *
	TokenDivide   TokenType = "DIVIDE"    // /
	TokenModulo   TokenType = "MODULO"    // %

	// Comparison operators
	TokenEquals       TokenType = "EQUALS"        // ==
	TokenNotEquals    TokenType = "NOT_EQUALS"    // !=
	TokenGreater      TokenType = "GREATER"       // >
	TokenLess         TokenType = "LESS"          // <
	TokenGreaterEqual TokenType = "GREATER_EQUAL" // >=
	TokenLessEqual    TokenType = "LESS_EQUAL"    // <=

	// Logical operators
	TokenAnd TokenType = "AND" // &&
	TokenOr  TokenType = "OR"  // ||
	TokenNot TokenType = "NOT" // !
)

// Token structure
type Token struct {
	Type     TokenType
	Value    string
	Path     string
	Line     int
	Column   int
}