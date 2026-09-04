package typechecker

import (
	"forst/internal/ast"
)

// bindClosureCaptures associates pending capture writes with the LHS variable
// after inferring a function literal (phase 4g).
func (tc *TypeChecker) bindClosureCaptures(lhs ast.ExpressionNode, _ ast.FunctionLiteralNode) {
	if tc == nil {
		return
	}
	vn, ok := lhs.(ast.VariableNode)
	if !ok {
		return
	}
	if tc.closureCaptures == nil {
		tc.closureCaptures = make(map[ast.Identifier][]*AccessPath)
	}
	writes := append([]*AccessPath(nil), tc.pendingClosureWrites...)
	tc.closureCaptures[vn.Ident.ID] = writes
	tc.pendingClosureWrites = nil
}

// applyClosureCallInvalidation drops facts when a captured-writing closure is called.
func (tc *TypeChecker) applyClosureCallInvalidation(fn ast.Identifier, span ast.SourceSpan) {
	if tc == nil || tc.closureCaptures == nil {
		return
	}
	writes, ok := tc.closureCaptures[fn]
	if !ok {
		return
	}
	for _, w := range writes {
		tc.invalidateOverlappingFactsWithReason(w, span, dropByConcurrent)
	}
}

// applyClosureEscapeInvalidation drops facts when a closure escapes to unknown code.
func (tc *TypeChecker) applyClosureEscapeInvalidation(arg ast.ExpressionNode, span ast.SourceSpan) {
	if tc == nil || arg == nil {
		return
	}
	vn, ok := arg.(ast.VariableNode)
	if !ok {
		return
	}
	tc.applyClosureCallInvalidation(vn.Ident.ID, span)
	// Also mark the capture roots as escaped on the current function summary.
	if writes, ok := tc.closureCaptures[vn.Ident.ID]; ok {
		for _, w := range writes {
			tc.recordEscapePattern(w)
		}
	}
}

// invalidateAfterSpawn drops facts on reference-bearing args of a go statement.
func (tc *TypeChecker) invalidateAfterSpawn(e ast.FunctionCallNode) {
	if tc == nil {
		return
	}
	span := e.CallSpan
	if !span.IsSet() {
		span = e.Function.Span
	}
	for _, arg := range e.Arguments {
		path := tc.accessPathForExpr(arg)
		if path == nil {
			continue
		}
		types, _ := tc.inferExpressionType(arg)
		may := true
		if len(types) == 1 {
			t := types[0]
			if t.Ident == ast.TypeInt || t.Ident == ast.TypeFloat || t.Ident == ast.TypeBool || t.Ident == ast.TypeString {
				may = false
			} else if !tc.TypeMayAlias(t) {
				may = false
			}
		}
		if !may {
			continue
		}
		root := tc.paths.Intern(AccessPath{Root: path.Root})
		tc.invalidateOverlappingFactsWithReason(root, span, dropByConcurrent)
		tc.recordSpawnPattern(path)
	}
}

// recordSpawnPattern notes a parameter path passed into a go statement on the current summary.
func (tc *TypeChecker) recordSpawnPattern(p *AccessPath) {
	if tc == nil || p == nil || tc.currentInferFn == "" {
		return
	}
	paramIdx, steps, ok := tc.paramIndexForPath(p)
	if !ok {
		return
	}
	tc.ensureSummaries()
	sum := tc.functionSummaries[tc.currentInferFn]
	if sum == nil {
		sum = &FunctionSummary{}
		tc.functionSummaries[tc.currentInferFn] = sum
	}
	sum.SpawnsWith = append(sum.SpawnsWith, AccessPattern{ParamIndex: paramIdx, Steps: steps})
}

// invalidateAfterChannelSend drops facts when sending reference-bearing values.
func (tc *TypeChecker) invalidateAfterChannelSend(value ast.ExpressionNode, span ast.SourceSpan) {
	if tc == nil || value == nil {
		return
	}
	path := tc.accessPathForExpr(value)
	if path == nil {
		return
	}
	types, _ := tc.inferExpressionType(value)
	if len(types) == 1 {
		t := types[0]
		if t.Ident == ast.TypeInt || t.Ident == ast.TypeFloat || t.Ident == ast.TypeBool || t.Ident == ast.TypeString {
			return
		}
		if !tc.TypeMayAlias(t) {
			return
		}
	}
	root := tc.paths.Intern(AccessPath{Root: path.Root})
	tc.invalidateOverlappingFactsWithReason(root, span, dropByConcurrent)
	tc.recordEscapePattern(path)
}
