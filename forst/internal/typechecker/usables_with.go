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

	var merged Ambient
	if len(tc.ambientStack) == 0 {
		merged = inner
	} else {
		outer := tc.ambientStack[len(tc.ambientStack)-1]
		merged = tc.mergeAmbient(outer, inner)
	}

	innerKeys := make(map[string]struct{}, len(inner.keys))
	for k := range inner.keys {
		innerKeys[k] = struct{}{}
	}
	tc.pendingWithChecks = append(tc.pendingWithChecks, pendingWithCheck{
		with:      with,
		innerKeys: innerKeys,
	})

	tc.ambientStack = append(tc.ambientStack, merged)
	tc.pushScope(with)

	for _, node := range with.Body {
		if _, err := tc.inferNodeType(node); err != nil {
			tc.popScope()
			tc.ambientStack = tc.ambientStack[:len(tc.ambientStack)-1]
			return nil, err
		}
	}

	tc.popScope()
	tc.ambientStack = tc.ambientStack[:len(tc.ambientStack)-1]
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
		tc.collectRequiredUsableRootsFromNode(node, required)
	}
	for key := range check.innerKeys {
		if _, needed := required[key]; !needed {
			tc.warnf(check.with.Span, "usables-unused-key",
				"wiring key %q is not required by any callee in this with-block", key)
		}
	}
}

func (tc *TypeChecker) collectRequiredUsableRootsFromNode(node ast.Node, out map[string]struct{}) {
	switch n := node.(type) {
	case ast.AssignmentNode:
		for _, rv := range n.RValues {
			tc.collectRequiredUsableRootsFromExpr(rv, out)
		}
	case ast.ReturnNode:
		for _, v := range n.Values {
			tc.collectRequiredUsableRootsFromExpr(v, out)
		}
	case ast.DeferNode:
		tc.collectRequiredUsableRootsFromExpr(n.Call, out)
	case *ast.DeferNode:
		if n != nil {
			tc.collectRequiredUsableRootsFromExpr(n.Call, out)
		}
	case ast.GoStmtNode:
		tc.collectRequiredUsableRootsFromExpr(n.Call, out)
	case *ast.GoStmtNode:
		if n != nil {
			tc.collectRequiredUsableRootsFromExpr(n.Call, out)
		}
	case ast.FunctionCallNode:
		tc.recordRequiredUsableRootsForCallee(n.Function.ID, out)
		for _, arg := range n.Arguments {
			tc.collectRequiredUsableRootsFromExpr(arg, out)
		}
	case ast.WithNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
	case ast.IfNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
		if n.Else != nil {
			tc.collectRequiredUsableRootsFromNode(*n.Else, out)
		}
	case *ast.IfNode:
		tc.collectRequiredUsableRootsFromNode(*n, out)
	case ast.ForNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
	case *ast.ForNode:
		for _, child := range n.Body {
			tc.collectRequiredUsableRootsFromNode(child, out)
		}
	case ast.EnsureNode:
		if n.Block != nil {
			for _, child := range n.Block.Body {
				tc.collectRequiredUsableRootsFromNode(child, out)
			}
		}
	}
}

func (tc *TypeChecker) collectRequiredUsableRootsFromExpr(expr ast.ExpressionNode, out map[string]struct{}) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case ast.FunctionCallNode:
		tc.recordRequiredUsableRootsForCallee(e.Function.ID, out)
		for _, arg := range e.Arguments {
			tc.collectRequiredUsableRootsFromExpr(arg, out)
		}
	case *ast.FunctionCallNode:
		if e != nil {
			tc.recordRequiredUsableRootsForCallee(e.Function.ID, out)
			for _, arg := range e.Arguments {
				tc.collectRequiredUsableRootsFromExpr(arg, out)
			}
		}
	case ast.BinaryExpressionNode:
		tc.collectRequiredUsableRootsFromExpr(e.Left, out)
		tc.collectRequiredUsableRootsFromExpr(e.Right, out)
	case ast.UnaryExpressionNode:
		tc.collectRequiredUsableRootsFromExpr(e.Operand, out)
	case ast.IndexExpressionNode:
		tc.collectRequiredUsableRootsFromExpr(e.Target, out)
		tc.collectRequiredUsableRootsFromExpr(e.Index, out)
	case ast.ReferenceNode:
		if v, ok := e.Value.(ast.ExpressionNode); ok {
			tc.collectRequiredUsableRootsFromExpr(v, out)
		}
	case ast.DereferenceNode:
		tc.collectRequiredUsableRootsFromValue(e.Value, out)
	case ast.ShapeNode:
		for _, field := range e.Fields {
			if v, ok := field.ValueExpression(); ok {
				tc.collectRequiredUsableRootsFromExpr(v, out)
			}
		}
	case ast.ArrayLiteralNode:
		for _, el := range e.Value {
			tc.collectRequiredUsableRootsFromValue(el, out)
		}
	case ast.MapLiteralNode:
		for _, entry := range e.Entries {
			tc.collectRequiredUsableRootsFromValue(entry.Key, out)
			tc.collectRequiredUsableRootsFromValue(entry.Value, out)
		}
	case ast.OkExprNode:
		tc.collectRequiredUsableRootsFromExpr(e.Value, out)
	case ast.ErrExprNode:
		tc.collectRequiredUsableRootsFromExpr(e.Value, out)
	case ast.AssertionNode:
		for _, c := range e.Constraints {
			for _, arg := range c.Args {
				if arg.Shape != nil {
					for _, field := range arg.Shape.Fields {
						if v, ok := field.ValueExpression(); ok {
							tc.collectRequiredUsableRootsFromExpr(v, out)
						}
					}
				}
				if arg.Value != nil {
					tc.collectRequiredUsableRootsFromValue(*arg.Value, out)
				}
			}
		}
	}
}

func (tc *TypeChecker) collectRequiredUsableRootsFromValue(v ast.ValueNode, out map[string]struct{}) {
	if v == nil {
		return
	}
	if expr, ok := v.(ast.ExpressionNode); ok {
		tc.collectRequiredUsableRootsFromExpr(expr, out)
	}
}

func (tc *TypeChecker) recordRequiredUsableRootsForCallee(callee ast.Identifier, out map[string]struct{}) {
	for _, slot := range tc.FunctionUsables[callee] {
		out[string(slot.RootIdent)] = struct{}{}
	}
}
