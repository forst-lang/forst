package typechecker

import (
	"fmt"
	"forst/internal/ast"
)

func (tc *TypeChecker) inferForNode(n *ast.ForNode) ([]ast.TypeNode, error) {
	tc.pushScope(n)
	defer tc.popScope()

	if !n.IsRange {
		if n.Init != nil {
			if _, err := tc.inferNodeType(n.Init); err != nil {
				return nil, err
			}
		}
		if n.Cond != nil {
			if _, err := tc.inferExpressionType(n.Cond); err != nil {
				return nil, err
			}
		}
		if n.Post != nil {
			if _, err := tc.inferNodeType(n.Post); err != nil {
				return nil, err
			}
		}
	} else {
		if err := tc.inferForRangeVars(n); err != nil {
			return nil, err
		}
	}

	tc.loopDepth++
	for _, stmt := range n.Body {
		if _, err := tc.inferNodeType(stmt); err != nil {
			tc.loopDepth--
			return nil, err
		}
	}
	tc.loopDepth--
	return nil, nil
}

func (tc *TypeChecker) inferForRangeVars(n *ast.ForNode) error {
	xt, err := tc.inferExpressionType(n.RangeX)
	if err != nil {
		return err
	}
	if len(xt) != 1 {
		return fmt.Errorf("range expression must have a single type, got %d", len(xt))
	}
	t := xt[0]

	noKey := n.RangeKey == nil
	noVal := n.RangeValue == nil
	if noKey && noVal {
		return nil
	}
	if !noKey && noVal {
		one, err := tc.rangeTypesForOneVar(t)
		if err != nil {
			return err
		}
		return tc.registerRangeBinding(n.RangeKey, one, n.RangeShort)
	}

	if noKey && !noVal {
		return fmt.Errorf("range loop value variable without key is invalid")
	}

	kTyp, vTyp, err := tc.rangeTypesForTwoVars(t)
	if err != nil {
		return err
	}
	if err := tc.registerRangeBinding(n.RangeKey, kTyp, n.RangeShort); err != nil {
		return err
	}
	return tc.registerRangeBinding(n.RangeValue, vTyp, n.RangeShort)
}

func (tc *TypeChecker) registerRangeBinding(id *ast.Ident, typ ast.TypeNode, isShort bool) error {
	if id == nil || id.ID == "_" {
		return nil
	}
	if isShort {
		tc.scopeStack.currentScope().RegisterSymbol(id.ID, []ast.TypeNode{typ}, SymbolVariable)
		tc.VariableTypes[id.ID] = []ast.TypeNode{typ}
		return nil
	}
	prev, ok := tc.scopeStack.LookupVariableType(id.ID)
	if !ok {
		return fmt.Errorf("undefined variable %q in range assignment", id.ID)
	}
	if len(prev) != 1 {
		return fmt.Errorf("cannot assign range value to %q", id.ID)
	}
	if prev[0].Ident != typ.Ident {
		return fmt.Errorf("range assignment type mismatch for %q", id.ID)
	}
	return nil
}

func (tc *TypeChecker) rangeTypesForOneVar(t ast.TypeNode) (ast.TypeNode, error) {
	switch t.Ident {
	case ast.TypeArray:
		if len(t.TypeParams) >= 1 {
			return ast.TypeNode{Ident: ast.TypeInt}, nil
		}
	case ast.TypeString:
		return ast.TypeNode{Ident: ast.TypeInt}, nil
	case ast.TypeMap:
		if len(t.TypeParams) >= 1 {
			return t.TypeParams[0], nil
		}
	}
	return ast.TypeNode{}, fmt.Errorf("unsupported range over type %s (single-variable form)", t.Ident)
}

func (tc *TypeChecker) rangeTypesForTwoVars(t ast.TypeNode) (keyT, valT ast.TypeNode, err error) {
	switch t.Ident {
	case ast.TypeArray:
		if len(t.TypeParams) >= 1 {
			return ast.TypeNode{Ident: ast.TypeInt}, t.TypeParams[0], nil
		}
	case ast.TypeMap:
		if len(t.TypeParams) >= 2 {
			return t.TypeParams[0], t.TypeParams[1], nil
		}
	case ast.TypeString:
		return ast.TypeNode{Ident: ast.TypeInt}, ast.TypeNode{Ident: ast.TypeInt}, nil
	}
	return ast.TypeNode{}, ast.TypeNode{}, fmt.Errorf("unsupported range over type %s (two-variable form)", t.Ident)
}
