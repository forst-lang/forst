package ast

// Compound assignment token identifiers.
const (
	TokenPlusEq       TokenIdent = "PLUS_EQ"        // +=
	TokenMinusEq      TokenIdent = "MINUS_EQ"       // -=
	TokenStarEq       TokenIdent = "STAR_EQ"        // *=
	TokenDivideEq     TokenIdent = "DIVIDE_EQ"      // /=
	TokenModuloEq     TokenIdent = "MODULO_EQ"      // %=
	TokenBitwiseAndEq TokenIdent = "BITWISE_AND_EQ" // &=
	TokenBitwiseOrEq  TokenIdent = "BITWISE_OR_EQ"  // |=
)

// IsCompoundAssignToken reports whether t is a compound assignment operator token.
func IsCompoundAssignToken(t TokenIdent) bool {
	switch t {
	case TokenPlusEq, TokenMinusEq, TokenStarEq, TokenDivideEq, TokenModuloEq,
		TokenBitwiseAndEq, TokenBitwiseOrEq:
		return true
	default:
		return false
	}
}

// IsAssignmentOperatorToken reports whether tok is =, :=, or a compound assignment operator.
func IsAssignmentOperatorToken(tok Token) bool {
	switch tok.Type {
	case TokenColonEquals:
		return true
	case TokenEquals:
		return tok.Value == "="
	default:
		return IsCompoundAssignToken(tok.Type)
	}
}

// CompoundAssignBinaryOp maps a compound assignment token to its binary operator.
func CompoundAssignBinaryOp(t TokenIdent) (TokenIdent, bool) {
	switch t {
	case TokenPlusEq:
		return TokenPlus, true
	case TokenMinusEq:
		return TokenMinus, true
	case TokenStarEq:
		return TokenStar, true
	case TokenDivideEq:
		return TokenDivide, true
	case TokenModuloEq:
		return TokenModulo, true
	case TokenBitwiseAndEq:
		return TokenBitwiseAnd, true
	case TokenBitwiseOrEq:
		return TokenBitwiseOr, true
	default:
		return "", false
	}
}

// CompoundAssignOperatorString returns the source spelling for a compound assignment token.
func CompoundAssignOperatorString(t TokenIdent) string {
	switch t {
	case TokenPlusEq:
		return "+="
	case TokenMinusEq:
		return "-="
	case TokenStarEq:
		return "*="
	case TokenDivideEq:
		return "/="
	case TokenModuloEq:
		return "%="
	case TokenBitwiseAndEq:
		return "&="
	case TokenBitwiseOrEq:
		return "|="
	default:
		return string(t)
	}
}
