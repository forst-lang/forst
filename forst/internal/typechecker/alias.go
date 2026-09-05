package typechecker

import (
	"forst/internal/ast"
)

// AliasContext owns points-to / may-alias state for refinement stability (phase 4d).
type AliasContext struct {
	// pathAliases maps an LHS path key to the storage paths it may name.
	// Strong update: reassignment replaces the set.
	pathAliases map[string][]*AccessPath
}

// newAliasContext creates empty points-to state for a TypeChecker.
func newAliasContext() *AliasContext {
	return &AliasContext{pathAliases: make(map[string][]*AccessPath)}
}

// ensureAliasCtx lazily allocates AliasContext on the TypeChecker.
func (tc *TypeChecker) ensureAliasCtx() *AliasContext {
	if tc.aliasCtx == nil {
		tc.aliasCtx = newAliasContext()
	}
	return tc.aliasCtx
}

// TypeMayAlias reports whether a value of typ may share mutable storage under Go copy rules.
func (tc *TypeChecker) TypeMayAlias(typ ast.TypeNode) bool {
	return tc.typeMayAlias(typ, map[ast.TypeIdent]bool{})
}

// typeMayAlias walks typ (and nested shapes) for reference-bearing storage; visiting breaks cycles.
func (tc *TypeChecker) typeMayAlias(typ ast.TypeNode, visiting map[ast.TypeIdent]bool) bool {
	switch typ.Ident {
	case ast.TypePointer, ast.TypeArray, ast.TypeMap, ast.TypeChannel, ast.TypeObject:
		return true
	case ast.TypeString, ast.TypeInt, ast.TypeFloat, ast.TypeBool, ast.TypeVoid, ast.TypeError,
		ast.TypeBytes:
		return false
	}
	if typ.IsFunctionType() {
		return true
	}
	if typ.Ident == ast.TypePointer || (len(typ.TypeParams) == 1 && typ.Ident == ast.TypePointer) {
		return true
	}
	if visiting[typ.Ident] {
		return true
	}
	visiting[typ.Ident] = true
	defer delete(visiting, typ.Ident)

	if shape, ok := tc.shapeFromTypeIdent(typ.Ident); ok {
		return tc.shapeMayAlias(shape, visiting)
	}
	return false
}

// shapeFromTypeIdent resolves a named typedef to its shape payload, if any.
func (tc *TypeChecker) shapeFromTypeIdent(id ast.TypeIdent) (*ast.ShapeNode, bool) {
	if tc == nil || id == "" {
		return nil, false
	}
	def, ok := tc.Defs[id]
	if !ok {
		return nil, false
	}
	switch d := def.(type) {
	case ast.TypeDefNode:
		return ast.PayloadShape(d.Expr)
	case *ast.TypeDefNode:
		if d != nil {
			return ast.PayloadShape(d.Expr)
		}
	}
	return nil, false
}

// shapeMayAlias is true if any field type or nested shape may alias under Go copy rules.
func (tc *TypeChecker) shapeMayAlias(shape *ast.ShapeNode, visiting map[ast.TypeIdent]bool) bool {
	if shape == nil {
		return false
	}
	for _, f := range shape.Fields {
		if f.Type != nil && tc.typeMayAlias(*f.Type, visiting) {
			return true
		}
		if f.Shape != nil && tc.shapeMayAlias(f.Shape, visiting) {
			return true
		}
	}
	return false
}

// recordAssignmentAlias updates alias sets for lhs := rhs (or =).
func (tc *TypeChecker) recordAssignmentAlias(lhs, rhs ast.ExpressionNode, rhsTypes []ast.TypeNode) {
	if tc == nil {
		return
	}

	ctx := tc.ensureAliasCtx()
	lhsPath := tc.accessPathForExpr(lhs)
	if lhsPath == nil {
		return
	}

	if tc.handleReferenceAssignment(lhsPath, rhs, ctx) {
		return
	}

	rhsPath := tc.accessPathForExpr(rhs)
	if rhsPath == nil {
		return
	}

	if tc.handleFieldExtractAssignment(lhsPath, rhsPath, ctx) {
		return
	}

	tc.handleWholeValueAssignment(lhsPath, rhsPath, rhsTypes, ctx)
}

// handleReferenceAssignment handles cases like p := &x.
func (tc *TypeChecker) handleReferenceAssignment(lhsPath *AccessPath, rhs ast.ExpressionNode, ctx *AliasContext) bool {
	ref, ok := rhs.(ast.ReferenceNode)
	if !ok {
		return false
	}
	inner := tc.accessPathForExpr(refValueExpr(ref))
	if inner != nil {
		ctx.strongUpdate(lhsPath, inner)
		tc.invalidateOverlappingFactsWithReason(inner, refSpan(ref), dropByConcurrent)
		tc.recordEscapePattern(inner)
	}
	return true
}

// handleFieldExtractAssignment handles cases like address := user.address.
func (tc *TypeChecker) handleFieldExtractAssignment(lhsPath, rhsPath *AccessPath, ctx *AliasContext) bool {
	if len(rhsPath.Steps) > 0 {
		ctx.strongUpdate(lhsPath, rhsPath)
		return true
	}
	return false
}

// handleWholeValueAssignment handles whole-value copy aliasing.
func (tc *TypeChecker) handleWholeValueAssignment(lhsPath, rhsPath *AccessPath, rhsTypes []ast.TypeNode, ctx *AliasContext) {
	if !typeSliceMayAlias(tc, rhsTypes) {
		delete(ctx.pathAliases, lhsPath.PathKey())
		return
	}
	ctx.strongUpdate(lhsPath, rhsPath)
}

// typeSliceMayAlias determines if any given type may alias.
func typeSliceMayAlias(tc *TypeChecker, types []ast.TypeNode) bool {
	for _, t := range types {
		if tc.TypeMayAlias(t) {
			return true
		}
	}
	return false
}

// strongUpdate replaces lhs's may-alias set with a single storage path (reassignment).
func (a *AliasContext) strongUpdate(lhs, storage *AccessPath) {
	if a == nil || lhs == nil || storage == nil {
		return
	}
	if a.pathAliases == nil {
		a.pathAliases = make(map[string][]*AccessPath)
	}
	a.pathAliases[lhs.PathKey()] = []*AccessPath{storage}
}

// expandWriteThroughAliases returns the write path plus any aliased storage paths
// with the same trailing steps after the alias root.
func (tc *TypeChecker) expandWriteThroughAliases(write *AccessPath) []*AccessPath {
	if write == nil {
		return nil
	}
	out := []*AccessPath{write}
	if tc.aliasCtx == nil || tc.paths == nil {
		return out
	}
	rootOnly := tc.paths.Intern(AccessPath{Root: write.Root})
	if bases, ok := tc.aliasCtx.pathAliases[rootOnly.PathKey()]; ok {
		for _, base := range bases {
			if base == nil {
				continue
			}
			mapped := tc.paths.Intern(AccessPath{
				Root:  base.Root,
				Steps: append(base.CloneSteps(), write.CloneSteps()...),
			})
			out = append(out, mapped)
		}
	}
	if bases, ok := tc.aliasCtx.pathAliases[write.PathKey()]; ok {
		out = append(out, bases...)
	}
	return out
}

// refValueExpr unwraps &x to the underlying expression x.
func refValueExpr(ref ast.ReferenceNode) ast.ExpressionNode {
	if ref.Value == nil {
		return nil
	}
	switch v := ref.Value.(type) {
	case ast.VariableNode:
		return v
	case *ast.VariableNode:
		if v != nil {
			return *v
		}
	case ast.ExpressionNode:
		return v
	}
	return nil
}

// refSpan returns the source span of a reference expression.
func refSpan(ref ast.ReferenceNode) ast.SourceSpan {
	if vn, ok := ref.Value.(ast.VariableNode); ok {
		return vn.Ident.Span
	}
	if vn, ok := ref.Value.(*ast.VariableNode); ok && vn != nil {
		return vn.Ident.Span
	}
	return ast.SourceSpan{}
}

// recordEscapePattern notes that a parameter path escapes (& / channel / closure) on the current summary.
func (tc *TypeChecker) recordEscapePattern(p *AccessPath) {
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
	sum.Escapes = append(sum.Escapes, AccessPattern{ParamIndex: paramIdx, Steps: steps})
}
