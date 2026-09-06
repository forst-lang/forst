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

	if n.Label != nil {
		tc.loopLabelStack = append(tc.loopLabelStack, n.Label.ID)
		defer func() { tc.loopLabelStack = tc.loopLabelStack[:len(tc.loopLabelStack)-1] }()
	}

	tc.loopDepth++
	tc.pushWriteCollector()
	for _, stmt := range n.Body {
		if _, err := tc.inferNodeType(stmt); err != nil {
			tc.popWriteCollectorAndInvalidate()
			tc.loopDepth--
			return nil, err
		}
	}
	tc.popWriteCollectorAndInvalidate()
	tc.loopDepth--
	return nil, nil
}

func (tc *TypeChecker) inferForRangeVars(n *ast.ForNode) error {
	rangeSpan := spanOfExpression(n.RangeX)
	xt, err := tc.inferExpressionType(n.RangeX)
	if err != nil {
		return err
	}
	if len(xt) != 1 {
		return reportf(rangeSpan, "range-expression-type",
			"range expression must have a single type",
			fmt.Sprintf("The range expression has %d types; `for range` needs exactly one.", len(xt)),
			"bind the collection to a single-typed variable first")
	}
	t := xt[0]

	noKey := n.RangeKey == nil
	noVal := n.RangeValue == nil
	if noKey && noVal {
		return nil
	}
	if !noKey && noVal {
		one, err := tc.rangeTypesForOneVar(t, rangeSpan)
		if err != nil {
			return err
		}
		return tc.registerRangeBinding(n.RangeKey, one, n.RangeShort, rangeSpan)
	}

	if noKey && !noVal {
		return reportf(rangeSpan, "range-key-missing",
			"range loop value variable without key is invalid",
			"A two-slot range loop needs both key and value variables (`for k, v := range xs`).",
			"add a key variable or use single-variable `for i := range xs`")
	}

	kTyp, vTyp, err := tc.rangeTypesForTwoVars(t, rangeSpan)
	if err != nil {
		return err
	}
	if err := tc.registerRangeBinding(n.RangeKey, kTyp, n.RangeShort, rangeSpan); err != nil {
		return err
	}
	return tc.registerRangeBinding(n.RangeValue, vTyp, n.RangeShort, rangeSpan)
}

func (tc *TypeChecker) registerRangeBinding(id *ast.Ident, typ ast.TypeNode, isShort bool, rangeSpan ast.SourceSpan) error {
	if id == nil || id.ID == "_" {
		return nil
	}
	span := firstSetSpan(id.Span, rangeSpan)
	if isShort {
		tc.scopeStack.currentScope().RegisterSymbol(id.ID, []ast.TypeNode{typ}, SymbolVariable)
		tc.VariableTypes[id.ID] = []ast.TypeNode{typ}
		return nil
	}
	prev, ok := tc.scopeStack.LookupVariableType(id.ID)
	if !ok {
		return reportf(span, "undefined-symbol",
			fmt.Sprintf("undefined variable `%s` in range assignment", id.ID),
			fmt.Sprintf("Variable `%s` is not declared before the range assignment.", id.ID),
			"declare it with `var` or use `:=` in the range header")
	}
	if len(prev) != 1 {
		return reportf(span, "range-assignment-type",
			fmt.Sprintf("cannot assign range value to `%s`", id.ID),
			fmt.Sprintf("Variable `%s` does not have a single type for range assignment.", id.ID),
			"give the variable an explicit type or use `:=`")
	}
	if prev[0].Ident != typ.Ident {
		return reportf(span, "range-assignment-type",
			fmt.Sprintf("range assignment type mismatch for `%s`", id.ID),
			fmt.Sprintf("Variable `%s` has type `%s`, but this range yields `%s`.", id.ID, formatTypeIdentForDiag(prev[0].Ident), formatTypeIdentForDiag(typ.Ident)),
			"change the variable type or iterate a compatible collection")
	}
	return nil
}

func (tc *TypeChecker) rangeTypesForOneVar(t ast.TypeNode, span ast.SourceSpan) (ast.TypeNode, error) {
	switch t.Ident {
	case "Seq":
		if len(t.TypeParams) >= 1 {
			return t.TypeParams[0], nil
		}
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
	return ast.TypeNode{}, reportf(span, "range-unsupported-type",
		fmt.Sprintf("unsupported range over type `%s`", formatTypeIdentForDiag(t.Ident)),
		fmt.Sprintf("Single-variable `for range` does not support type `%s`.", formatTypeIdentForDiag(t.Ident)),
		"use a slice, map, string, or Seq collection")
}

func (tc *TypeChecker) rangeTypesForTwoVars(t ast.TypeNode, span ast.SourceSpan) (keyT, valT ast.TypeNode, err error) {
	switch t.Ident {
	case "Seq":
		if len(t.TypeParams) >= 1 {
			return ast.TypeNode{Ident: ast.TypeInt}, t.TypeParams[0], nil
		}
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
	return ast.TypeNode{}, ast.TypeNode{}, reportf(span, "range-unsupported-type",
		fmt.Sprintf("unsupported range over type `%s`", formatTypeIdentForDiag(t.Ident)),
		fmt.Sprintf("Two-variable `for k, v := range` does not support type `%s`.", formatTypeIdentForDiag(t.Ident)),
		"use a slice, map, string, or Seq collection")
}

func (tc *TypeChecker) hasLoopLabel(label ast.Identifier) bool {
	for _, l := range tc.loopLabelStack {
		if l == label {
			return true
		}
	}
	return false
}
