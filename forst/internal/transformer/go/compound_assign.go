package transformergo

import (
	"fmt"
	"go/token"

	"forst/internal/ast"
)

func compoundAssignGoToken(op ast.TokenIdent) (token.Token, bool) {
	switch op {
	case ast.TokenPlusEq:
		return token.ADD_ASSIGN, true
	case ast.TokenMinusEq:
		return token.SUB_ASSIGN, true
	case ast.TokenStarEq:
		return token.MUL_ASSIGN, true
	case ast.TokenDivideEq:
		return token.QUO_ASSIGN, true
	case ast.TokenModuloEq:
		return token.REM_ASSIGN, true
	case ast.TokenBitwiseAndEq:
		return token.AND_ASSIGN, true
	case ast.TokenBitwiseOrEq:
		return token.OR_ASSIGN, true
	default:
		return 0, false
	}
}

func assignmentGoToken(s ast.AssignmentNode) (token.Token, error) {
	if s.IsShort {
		return token.DEFINE, nil
	}
	if s.CompoundOp != "" {
		op, ok := compoundAssignGoToken(s.CompoundOp)
		if !ok {
			return 0, fmt.Errorf("unsupported compound assignment operator %s", s.CompoundOp)
		}
		return op, nil
	}
	return token.ASSIGN, nil
}
