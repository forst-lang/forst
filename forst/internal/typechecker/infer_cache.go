package typechecker

import (
	"forst/internal/ast"
)

// Package-level inference cache helpers. lookupCachedExpressionTypes reads tc.Types
// populated by storeInferredType; isLiteralExpression identifies hash-stable literals.
// Wiring into inferExpressionType is limited to literals: index/map reads and assign
// targets share structural hashes but differ in inferred types.
// VariableNode is excluded at call sites: occurrence narrowing uses span keys in
// variableOccurrenceTypes, not structural hash alone (see storeInferredType).
func (tc *TypeChecker) lookupCachedExpressionTypes(expr ast.Node) ([]ast.TypeNode, bool, error) {
	types, err := tc.LookupInferredType(expr, false)
	if err != nil {
		return nil, false, err
	}
	if types == nil {
		return nil, false, nil
	}
	return append([]ast.TypeNode(nil), types...), true, nil
}

// isLiteralExpression reports whether expr is a literal with a fixed builtin type.
func isLiteralExpression(expr ast.Node) bool {
	switch expr.(type) {
	case ast.IntLiteralNode, ast.FloatLiteralNode, ast.StringLiteralNode, ast.BoolLiteralNode, ast.NilLiteralNode:
		return true
	default:
		return false
	}
}
