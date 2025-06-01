package ast

// TokenIdent is the type for token identifiers
type TokenIdent string

const (
	// TokenImport is the token for import statements
	TokenImport TokenIdent = "IMPORT"
	// TokenPackage is the token for package statements
	TokenPackage TokenIdent = "PACKAGE"

	// TokenIdentifier is the token for user-defined identifiers
	TokenIdentifier TokenIdent = "IDENTIFIER"

	// TokenFunc is the token for function keyword
	TokenFunc TokenIdent = "FUNCTION"
	// TokenType is the token for type keyword
	TokenType TokenIdent = "TYPE"

	// TokenInt is the token for built-in int type
	TokenInt TokenIdent = "TYPE_INT"
	// TokenFloat is the token for built-in float type
	TokenFloat TokenIdent = "TYPE_FLOAT"
	// TokenString is the token for built-in string type
	TokenString TokenIdent = "TYPE_STRING"
	// TokenBool is the token for built-in bool type
	TokenBool TokenIdent = "TYPE_BOOL"
	// TokenArray is the token for built-in array type
	TokenArray TokenIdent = "TYPE_ARRAY"
	// TokenStruct is the token for built-in struct type
	TokenStruct TokenIdent = "TYPE_STRUCT"
	// TokenVoid is the token for built-in void type
	TokenVoid TokenIdent = "TYPE_VOID"

	// TokenIntLiteral is the token for integer literals
	TokenIntLiteral TokenIdent = "INT_LITERAL"
	// TokenFloatLiteral is the token for float literals
	TokenFloatLiteral TokenIdent = "FLOAT_LITERAL"
	// TokenStringLiteral is the token for string literals
	TokenStringLiteral TokenIdent = "STRING_LITERAL"
	// TokenTrue is the token for true boolean literal
	TokenTrue TokenIdent = "TRUE_LITERAL"
	// TokenFalse is the token for false boolean literal
	TokenFalse TokenIdent = "FALSE_LITERAL"

	// TokenLParen is the token for left parenthesis
	TokenLParen TokenIdent = "LPAREN"
	// TokenRParen is the token for right parenthesis
	TokenRParen TokenIdent = "RPAREN"
	// TokenLBrace is the token for left brace
	TokenLBrace TokenIdent = "LBRACE"
	// TokenRBrace is the token for right brace
	TokenRBrace TokenIdent = "RBRACE"
	// TokenLBracket is the token for left bracket
	TokenLBracket TokenIdent = "LBRACKET"
	// TokenRBracket is the token for right bracket
	TokenRBracket TokenIdent = "RBRACKET"
	// TokenReturn is the token for return keyword
	TokenReturn TokenIdent = "RETURN"
	// TokenEnsure is the token for ensure keyword
	TokenEnsure TokenIdent = "ENSURE"
	// TokenIs is the token for is keyword
	TokenIs TokenIdent = "IS"
	// TokenOr is the token for or keyword
	TokenOr TokenIdent = "OR"
	// TokenEOF is the token for end of file
	TokenEOF TokenIdent = "EOF"
	// TokenColon is the token for colon
	TokenColon TokenIdent = "COLON"
	// TokenComma is the token for comma
	TokenComma TokenIdent = "COMMA"
	// TokenDot is the token for dot
	TokenDot TokenIdent = "DOT"
	// TokenColonEquals is the token for :=
	TokenColonEquals TokenIdent = "COLON_EQUALS"
	// TokenSemicolon is the token for semicolon
	TokenSemicolon TokenIdent = "SEMICOLON"
	// TokenPlusPlus is the token for increment operator
	TokenPlusPlus TokenIdent = "PLUS_PLUS" // ++
	// TokenMinusMinus is the token for decrement operator
	TokenMinusMinus TokenIdent = "MINUS_MINUS" // --

	// TokenPlus is the token for plus operator
	TokenPlus TokenIdent = "PLUS" // +
	// TokenMinus is the token for minus operator
	TokenMinus TokenIdent = "MINUS" // -
	// TokenStar is the token for multiply operator, also used for pointer dereference
	TokenStar TokenIdent = "STAR" // *
	// TokenDivide is the token for divide operator
	TokenDivide TokenIdent = "DIVIDE" // /
	// TokenModulo is the token for modulo operator
	TokenModulo TokenIdent = "MODULO" // %

	// TokenEquals is the token for equals operator
	TokenEquals TokenIdent = "EQUALS" // ==
	// TokenNotEquals is the token for not equals operator
	TokenNotEquals TokenIdent = "NOT_EQUALS" // !=
	// TokenGreater is the token for greater than operator
	TokenGreater TokenIdent = "GREATER" // >
	// TokenLess is the token for less than operator
	TokenLess TokenIdent = "LESS" // <
	// TokenGreaterEqual is the token for greater or equal operator
	TokenGreaterEqual TokenIdent = "GREATER_EQUAL" // >=
	// TokenLessEqual is the token for less or equal operator
	TokenLessEqual TokenIdent = "LESS_EQUAL" // <=

	// TokenLogicalAnd is the token for logical and operator
	TokenLogicalAnd TokenIdent = "LOGICAL_AND" // &&
	// TokenLogicalOr is the token for logical or operator
	TokenLogicalOr TokenIdent = "LOGICAL_OR" // ||
	// TokenLogicalNot is the token for logical not operator
	TokenLogicalNot TokenIdent = "LOGICAL_NOT" // !

	// TokenBitwiseAnd is the token for bitwise and operator
	TokenBitwiseAnd TokenIdent = "BITWISE_AND" // &
	// TokenBitwiseOr is the token for bitwise or operator
	TokenBitwiseOr TokenIdent = "BITWISE_OR" // |

	// TokenComment is the token for comments
	TokenComment TokenIdent = "COMMENT" // //

	// TokenIf is the token for if statement
	TokenIf TokenIdent = "IF"
	// TokenElseIf is the token for else-if statement
	TokenElseIf TokenIdent = "ELSE_IF"
	// TokenElse is the token for else statement
	TokenElse TokenIdent = "ELSE"
	// TokenFor is the token for for loop
	TokenFor TokenIdent = "FOR"
	// TokenRange is the token for range keyword
	TokenRange TokenIdent = "RANGE"
	// TokenBreak is the token for break statement
	TokenBreak TokenIdent = "BREAK"
	// TokenContinue is the token for continue statement
	TokenContinue TokenIdent = "CONTINUE"
	// TokenSwitch is the token for switch statement
	TokenSwitch TokenIdent = "SWITCH"
	// TokenCase is the token for case statement
	TokenCase TokenIdent = "CASE"
	// TokenDefault is the token for default case
	TokenDefault TokenIdent = "DEFAULT"
	// TokenFallthrough is the token for fallthrough statement
	TokenFallthrough TokenIdent = "FALLTHROUGH"

	// TokenVar is the token for var keyword
	TokenVar TokenIdent = "VAR"
	// TokenConst is the token for const keyword
	TokenConst TokenIdent = "CONST"
	// TokenMap is the token for map keyword
	TokenMap TokenIdent = "MAP"
	// TokenChan is the token for chan keyword
	TokenChan TokenIdent = "CHAN"
	// TokenArrow is the token for arrow operator
	TokenArrow TokenIdent = "ARROW"
	// TokenInterface is the token for interface keyword
	TokenInterface TokenIdent = "INTERFACE"
	// TokenGo is the token for go keyword
	TokenGo TokenIdent = "GO"
	// TokenDefer is the token for defer keyword
	TokenDefer TokenIdent = "DEFER"
	// TokenGoto is the token for goto keyword
	TokenGoto TokenIdent = "GOTO"
)

// Token structure
type Token struct {
	Type   TokenIdent
	Value  string
	Path   string
	Line   int
	Column int
}

// IsBinaryOperator returns true if the token is a binary operator
func (t TokenIdent) IsBinaryOperator() bool {
	return t == TokenPlus || t == TokenMinus ||
		t == TokenStar || t == TokenDivide ||
		t == TokenModulo || t == TokenEquals ||
		t == TokenNotEquals || t == TokenGreater ||
		t == TokenLess || t == TokenGreaterEqual ||
		t == TokenLessEqual || t == TokenLogicalAnd ||
		t == TokenLogicalOr
}

// IsUnaryOperator returns true if the token is a unary operator
func (t TokenIdent) IsUnaryOperator() bool {
	return t == TokenLogicalNot
}

// IsLiteral returns true if the token is a literal
func (t TokenIdent) IsLiteral() bool {
	return t == TokenIntLiteral || t == TokenFloatLiteral || t == TokenStringLiteral || t == TokenTrue || t == TokenFalse
}

// IsArithmeticBinaryOperator returns true if the token is an arithmetic binary operator
func (t TokenIdent) IsArithmeticBinaryOperator() bool {
	return t == TokenPlus || t == TokenMinus || t == TokenStar || t == TokenDivide || t == TokenModulo
}

// IsComparisonBinaryOperator returns true if the token is a comparison binary operator
func (t TokenIdent) IsComparisonBinaryOperator() bool {
	return t == TokenEquals || t == TokenNotEquals || t == TokenGreater || t == TokenLess || t == TokenGreaterEqual || t == TokenLessEqual
}

// IsLogicalBinaryOperator returns true if the token is a logical binary operator
func (t TokenIdent) IsLogicalBinaryOperator() bool {
	return t == TokenLogicalAnd || t == TokenLogicalOr
}

func (t TokenIdent) String() string {
	switch t {
	case TokenBitwiseAnd:
		return "&"
	case TokenBitwiseOr:
		return "|"
	case TokenLogicalAnd:
		return "&&"
	case TokenLogicalOr:
		return "||"
	}
	return string(t)
}
