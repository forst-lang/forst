package typechecker

import (
	"fmt"

	"forst/internal/ast"
	"forst/internal/astwalk"
)

func (tc *TypeChecker) inferWithNode(node ast.Node) ([]ast.TypeNode, error) {
	with, ok := node.(ast.WithNode)
	if !ok {
		return nil, fmt.Errorf("inferWithNode: unexpected node type %T", node)
	}
	eng := tc.providersEngine()
	inner, err := tc.ambientFromWiringExpr(with.Wiring, with.Span)
	if err != nil {
		return nil, err
	}

	var merged ProviderScope
	if len(eng.ScopeStack) == 0 {
		merged = inner
	} else {
		outer := eng.ScopeStack[len(eng.ScopeStack)-1]
		merged = tc.mergeProviderScope(outer, inner)
	}

	innerKeys := make(map[string]struct{}, len(inner.Keys))
	for k := range inner.Keys {
		innerKeys[k] = struct{}{}
	}
	eng.PendingWith = append(eng.PendingWith, pendingWithCheck{
		with:      with,
		innerKeys: innerKeys,
	})

	eng.ScopeStack = append(eng.ScopeStack, merged)
	tc.pushScope(node)

	for _, node := range with.Body {
		if _, err := tc.inferNodeType(node); err != nil {
			tc.popScope()
			eng.ScopeStack = eng.ScopeStack[:len(eng.ScopeStack)-1]
			return nil, err
		}
	}

	tc.popScope()
	eng.ScopeStack = eng.ScopeStack[:len(eng.ScopeStack)-1]
	return nil, nil
}

func (tc *TypeChecker) warnf(span ast.SourceSpan, code, format string, a ...any) {
	tc.Warnings = append(tc.Warnings, Diagnostic{
		Msg:  fmt.Sprintf(format, a...),
		Span: span,
		Code: code,
	})
}

func (tc *TypeChecker) checkUnusedWiringKeys(check pendingWithCheck) {
	required := make(map[string]struct{})
	tc.collectRequiredProviderRootsFromStmts(check.with.Body, required)
	for key := range check.innerKeys {
		if _, needed := required[key]; !needed {
			tc.warnf(check.with.Span, "providers-unused-key",
				"wiring key %q is not required by any callee in this with-block", key)
		}
	}
}

func (tc *TypeChecker) collectRequiredProviderRootsFromStmts(stmts []ast.Node, out map[string]struct{}) {
	astwalk.WalkStmts(stmts, astwalk.StmtVisitor{
		OnWith: func(w ast.WithNode) bool {
			astwalk.WalkStmts(w.Body, astwalk.StmtVisitor{
				OnCall: tc.callCollector(out),
			})
			return false
		},
		OnIf: func(ifn ast.IfNode) bool {
			tc.collectRequiredProviderRootsFromStmts(ifn.Body, out)
			for _, branch := range ifn.ElseIfs {
				tc.collectRequiredProviderRootsFromStmts(branch.Body, out)
			}
			if ifn.Else != nil {
				tc.collectRequiredProviderRootsFromStmts(ifn.Else.Body, out)
			}
			return false
		},
		OnFor: func(forN ast.ForNode) bool {
			tc.collectRequiredProviderRootsFromStmts(forN.Body, out)
			return false
		},
		OnEnsure: func(e ast.EnsureNode) bool {
			if e.Block != nil {
				tc.collectRequiredProviderRootsFromStmts(e.Block.Body, out)
			}
			return false
		},
		OnCall: tc.callCollector(out),
	})
}

func (tc *TypeChecker) callCollector(out map[string]struct{}) func(ast.FunctionCallNode) bool {
	return func(call ast.FunctionCallNode) bool {
		tc.recordRequiredProviderRootsForCallee(call.Function.ID, out)
		astwalk.WalkExpr(call, astwalk.ExprVisitor{
			OnCall: func(inner ast.FunctionCallNode) bool {
				tc.recordRequiredProviderRootsForCallee(inner.Function.ID, out)
				return true
			},
		})
		return true
	}
}

func (tc *TypeChecker) recordRequiredProviderRootsForCallee(callee ast.Identifier, out map[string]struct{}) {
	for _, slot := range tc.providerSlotsForCallee(callee) {
		out[string(slot.RootIdent)] = struct{}{}
	}
}
