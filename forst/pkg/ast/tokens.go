package ast

const (
	TokenFunc      = "FUNC"
	TokenIdent     = "IDENT"
	TokenArrow     = "ARROW"
	TokenLParen    = "LPAREN"
	TokenRParen    = "RPAREN"
	TokenLBrace    = "LBRACE"
	TokenRBrace    = "RBRACE"
	TokenReturn    = "RETURN"
	TokenString    = "STRING"
	TokenInt       = "INT"
	TokenAssert    = "ASSERT"
	TokenEOF       = "EOF"

	// Arithmetic operators
	TokenPlus     = "PLUS"      // +
	TokenMinus    = "MINUS"     // -
	TokenMultiply = "MULTIPLY"  // *
	TokenDivide   = "DIVIDE"    // /
	TokenModulo   = "MODULO"    // %

	// Comparison operators
	TokenEquals       = "EQUALS"        // ==
	TokenNotEquals    = "NOT_EQUALS"    // !=
	TokenGreater      = "GREATER"       // >
	TokenLess         = "LESS"          // <
	TokenGreaterEqual = "GREATER_EQUAL" // >=
	TokenLessEqual    = "LESS_EQUAL"    // <=

	// Logical operators
	TokenAnd = "AND" // &&
	TokenOr  = "OR"  // ||
	TokenNot = "NOT" // !
)

// Token structure
type Token struct {
	Type     string
	Value    string
	Path 	 string
	Line     int
	Column   int
}