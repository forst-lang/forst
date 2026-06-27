package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

func (tc *TypeChecker) inferWithNode(with ast.WithNode) ([]ast.TypeNode, error) {
	inner, err := tc.ambientFromWiringExpr(with.Wiring, with.Span)
	if err != nil {
		return nil, err
	}

	var merged ProviderScope
	if len(tc.providerScopeStack) == 0 {
		merged = inner
	} else {
		outer := tc.providerScopeStack[len(tc.providerScopeStack)-1]
		merged = tc.mergeProviderScope(outer, inner)
	}

	innerKeys := make(map[string]struct{}, len(inner.keys))
	for k := range inner.keys {
		innerKeys[k] = struct{}{}
	}
	tc.pendingWithChecks = append(tc.pendingWithChecks, pendingWithCheck{
		with:      with,
		innerKeys: innerKeys,
	})

	tc.providerScopeStack = append(tc.providerScopeStack, merged)
	tc.pushScope(with)

	for _, node := range with.Body {
		if _, err := tc.inferNodeType(node); err != nil {
			tc.popScope()
			tc.providerScopeStack = tc.providerScopeStack[:len(tc.providerScopeStack)-1]
			return nil, err
		}
	}

	tc.popScope()
	tc.providerScopeStack = tc.providerScopeStack[:len(tc.providerScopeStack)-1]
	return nil, nil
}

func (tc *TypeChecker) warnf(span ast.SourceSpan, code, format string, a ...interface{}) {
	tc.Warnings = append(tc.Warnings, Diagnostic{
		Msg:  fmt.Sprintf(format, a...),
		Span: span,
		Code: code,
	})
}

func (tc *TypeChecker) checkUnusedWiringKeys(check pendingWithCheck) {
	required := make(map[string]struct{})
	for _, node := range check.with.Body {
		tc.collectRequiredProviderRootsFromNode(node, required)
	}
	for key := range check.innerKeys {
		if _, needed := required[key]; !needed {
			tc.warnf(check.with.Span, "providers-unused-key",
				"wiring key %q is not required by any callee in this with-block", key)
		}
	}
}

func (tc *TypeChecker) collectRequiredProviderRootsFromNode(node ast.Node, out map[string]struct{}) {
	switch n := node.(type) {
	case ast.AssignmentNode:
		for _, rv := range n.RValues {
			tc.collectRequiredProviderRootsFromExpr(rv, out)
		}
	case ast.ReturnNode:
		for _, v := range n.Values {
			tc.collectRequiredProviderRootsFromExpr(v, out)
		}
	case ast.DeferNode:
		tc.collectRequiredProviderRootsFromExpr(n.Call, out)
	case *ast.DeferNode:
		if n != nil {
			tc.collectRequiredProviderRootsFromExpr(n.Call, out)
		}
	case ast.GoStmtNode:
		tc.collectRequiredProviderRootsFromExpr(n.Call, out)
	case *ast.GoStmtNode:
		if n != nil {
			tc.collectRequiredProviderRootsFromExpr(n.Call, out)
		}
	case ast.FunctionCallNode:
		tc.recordRequiredProviderRootsForCallee(n.Function.ID, out)
		for _, arg := range n.Arguments {
			tc.collectRequiredProviderRootsFromExpr(arg, out)
		}
	case ast.WithNode:
		for _, child := range n.Body {
			tc.collectRequiredProviderRootsFromNode(child, out)
		}
	case ast.IfNode:
		for _, child := range n.Body {
			tc.collectRequiredProviderRootsFromNode(child, out)
		}
		if n.Else != nil {
			tc.collectRequiredProviderRootsFromNode(*n.Else, out)
		}
	case *ast.IfNode:
		tc.collectRequiredProviderRootsFromNode(*n, out)
	case ast.ForNode:
		for _, child := range n.Body {
			tc.collectRequiredProviderRootsFromNode(child, out)
		}
	case *ast.ForNode:
		for _, child := range n.Body {
			tc.collectRequiredProviderRootsFromNode(child, out)
		}
	case ast.EnsureNode:
		if n.Block != nil {
			for _, child := range n.Block.Body {
				tc.collectRequiredProviderRootsFromNode(child, out)
			}
		}
	}
}

func (tc *TypeChecker) collectRequiredProviderRootsFromExpr(expr ast.ExpressionNode, out map[string]struct{}) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case ast.FunctionCallNode:
		tc.recordRequiredProviderRootsForCallee(e.Function.ID, out)
		for _, arg := range e.Arguments {
			tc.collectRequiredProviderRootsFromExpr(arg, out)
		}
	case *ast.FunctionCallNode:
		if e != nil {
			tc.recordRequiredProviderRootsForCallee(e.Function.ID, out)
			for _, arg := range e.Arguments {
				tc.collectRequiredProviderRootsFromExpr(arg, out)
			}
		}
	case ast.BinaryExpressionNode:
		tc.collectRequiredProviderRootsFromExpr(e.Left, out)
		tc.collectRequiredProviderRootsFromExpr(e.Right, out)
	case ast.UnaryExpressionNode:
		tc.collectRequiredProviderRootsFromExpr(e.Operand, out)
	case ast.IndexExpressionNode:
		tc.collectRequiredProviderRootsFromExpr(e.Target, out)
		tc.collectRequiredProviderRootsFromExpr(e.Index, out)
	case ast.ReferenceNode:
		if v, ok := e.Value.(ast.ExpressionNode); ok {
			tc.collectRequiredProviderRootsFromExpr(v, out)
		}
	case ast.DereferenceNode:
		tc.collectRequiredProviderRootsFromValue(e.Value, out)
	case ast.ShapeNode:
		for _, field := range e.Fields {
			if v, ok := field.ValueExpression(); ok {
				tc.collectRequiredProviderRootsFromExpr(v, out)
			}
		}
	case ast.ArrayLiteralNode:
		for _, el := range e.Value {
			tc.collectRequiredProviderRootsFromValue(el, out)
		}
	case ast.MapLiteralNode:
		for _, entry := range e.Entries {
			tc.collectRequiredProviderRootsFromValue(entry.Key, out)
			tc.collectRequiredProviderRootsFromValue(entry.Value, out)
		}
	case ast.OkExprNode:
		tc.collectRequiredProviderRootsFromExpr(e.Value, out)
	case ast.ErrExprNode:
		tc.collectRequiredProviderRootsFromExpr(e.Value, out)
	case ast.AssertionNode:
		for _, c := range e.Constraints {
			for _, arg := range c.Args {
				if arg.Shape != nil {
					for _, field := range arg.Shape.Fields {
						if v, ok := field.ValueExpression(); ok {
							tc.collectRequiredProviderRootsFromExpr(v, out)
						}
					}
				}
				if arg.Value != nil {
					tc.collectRequiredProviderRootsFromValue(*arg.Value, out)
				}
			}
		}
	}
}

func (tc *TypeChecker) collectRequiredProviderRootsFromValue(v ast.ValueNode, out map[string]struct{}) {
	if v == nil {
		return
	}
	if expr, ok := v.(ast.ExpressionNode); ok {
		tc.collectRequiredProviderRootsFromExpr(expr, out)
	}
}

func (tc *TypeChecker) recordRequiredProviderRootsForCallee(callee ast.Identifier, out map[string]struct{}) {
	for _, slot := range tc.FunctionProviders[callee] {
		out[string(slot.RootIdent)] = struct{}{}
	}
}
