package transformergo

import (
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// negateCondition negates a boolean expression in place when possible
// (`err == nil` → `err != nil`, `speed < 100` → `speed >= 100`).
// Falls back to a leading NOT only for calls and other non-relational forms.
func negateCondition(condition goast.Expr) goast.Expr {
	switch e := condition.(type) {
	case *goast.ParenExpr:
		return negateCondition(e.X)
	case *goast.UnaryExpr:
		if e.Op == token.NOT {
			return e.X
		}
	case *goast.BinaryExpr:
		if inv, ok := invertRelOp(e.Op); ok {
			return &goast.BinaryExpr{X: e.X, Op: inv, Y: e.Y}
		}
		if e.Op == token.LAND {
			return &goast.BinaryExpr{X: negateCondition(e.X), Op: token.LOR, Y: negateCondition(e.Y)}
		}
		if e.Op == token.LOR {
			return &goast.BinaryExpr{X: negateCondition(e.X), Op: token.LAND, Y: negateCondition(e.Y)}
		}
	}
	return &goast.UnaryExpr{Op: token.NOT, X: condition}
}

func invertRelOp(op token.Token) (token.Token, bool) {
	switch op {
	case token.EQL:
		return token.NEQ, true
	case token.NEQ:
		return token.EQL, true
	case token.LSS:
		return token.GEQ, true
	case token.GTR:
		return token.LEQ, true
	case token.LEQ:
		return token.GTR, true
	case token.GEQ:
		return token.LSS, true
	default:
		return 0, false
	}
}

// disjoin joins a list of conditions with OR ("any condition must match")
func disjoin(conditions []goast.Expr) goast.Expr {
	if len(conditions) == 0 {
		return &goast.Ident{Name: BoolConstantFalse}
	}
	combined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined = &goast.BinaryExpr{
			X:  combined,
			Op: token.LOR,
			Y:  conditions[i],
		}
	}
	return combined
}

// conjoin joins a list of conditions with AND ("all conditions must match")
func conjoin(conditions []goast.Expr) goast.Expr {
	if len(conditions) == 0 {
		return &goast.Ident{Name: BoolConstantFalse}
	}
	combined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined = &goast.BinaryExpr{
			X:  combined,
			Op: token.LAND,
			Y:  conditions[i],
		}
	}
	return combined
}

func (t *Transformer) transformOperator(op ast.TokenIdent) (token.Token, error) {
	switch op {
	case ast.TokenPlus:
		return token.ADD, nil
	case ast.TokenMinus:
		return token.SUB, nil
	case ast.TokenStar:
		return token.MUL, nil
	case ast.TokenDivide:
		return token.QUO, nil
	case ast.TokenModulo:
		return token.REM, nil
	case ast.TokenEquals:
		return token.EQL, nil
	case ast.TokenNotEquals:
		return token.NEQ, nil
	case ast.TokenGreater:
		return token.GTR, nil
	case ast.TokenLess:
		return token.LSS, nil
	case ast.TokenGreaterEqual:
		return token.GEQ, nil
	case ast.TokenLessEqual:
		return token.LEQ, nil
	case ast.TokenLogicalAnd:
		return token.LAND, nil
	case ast.TokenLogicalOr:
		return token.LOR, nil
	case ast.TokenLogicalNot:
		return token.NOT, nil
	case ast.TokenBitwiseAnd:
		return token.AND, nil
	case ast.TokenBitwiseOr:
		return token.OR, nil
	case ast.TokenXor:
		return token.XOR, nil
	case ast.TokenLShift:
		return token.SHL, nil
	case ast.TokenRShift:
		return token.SHR, nil
	case ast.TokenAndNot:
		return token.AND_NOT, nil
	}

	return 0, fmt.Errorf("unsupported operator: %s", op)
}
