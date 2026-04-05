package printer

import (
	"fmt"

	"forst/internal/ast"
)

func tokenBinary(op ast.TokenIdent) string {
	switch op {
	case ast.TokenPlus:
		return "+"
	case ast.TokenMinus:
		return "-"
	case ast.TokenStar:
		return "*"
	case ast.TokenDivide:
		return "/"
	case ast.TokenModulo:
		return "%"
	case ast.TokenEquals:
		return "=="
	case ast.TokenNotEquals:
		return "!="
	case ast.TokenGreater:
		return ">"
	case ast.TokenLess:
		return "<"
	case ast.TokenGreaterEqual:
		return ">="
	case ast.TokenLessEqual:
		return "<="
	case ast.TokenLogicalAnd:
		return "&&"
	case ast.TokenLogicalOr:
		return "||"
	case ast.TokenIs:
		return "is"
	case ast.TokenBitwiseAnd:
		return "&"
	case ast.TokenBitwiseOr:
		return "|"
	default:
		return string(op)
	}
}

func tokenUnary(op ast.TokenIdent) string {
	switch op {
	case ast.TokenLogicalNot:
		return "!"
	case ast.TokenMinus:
		return "-"
	case ast.TokenStar:
		return "*"
	case ast.TokenBitwiseAnd:
		return "&"
	default:
		return string(op)
	}
}

func (p *printer) printExprFromNode(n ast.Node) (string, error) {
	if n == nil {
		return "", fmt.Errorf("printer: nil expression node")
	}
	e, ok := n.(ast.ExpressionNode)
	if !ok {
		return "", fmt.Errorf("printer: expected expression, got %T", n)
	}
	return p.printExpr(e)
}
