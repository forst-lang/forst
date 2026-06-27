package typechecker

import (
	"fmt"

	"forst/internal/ast"
	"forst/internal/providersgraph"
)

func (tc *TypeChecker) currentMergedScope() map[string]ast.TypeNode {
	eng := tc.providersEngine()
	return providersgraph.MergeScopeStack(eng.ScopeStack)
}

func (tc *TypeChecker) mergeProviderScope(outer, inner ProviderScope) ProviderScope {
	return providersgraph.MergeProviderScope(outer, inner)
}

func (tc *TypeChecker) providerScopeFromShape(shape ast.ShapeNode) (ProviderScope, error) {
	eng := tc.providersEngine()
	amb := newProviderScope()
	for fieldName, field := range shape.Fields {
		if field.IsMethod {
			continue
		}
		fieldSpan := shapeFieldSpan(shape, fieldName, field)
		if err := tc.validateWiringKey(fieldName, fieldSpan); err != nil {
			return ProviderScope{}, err
		}
		expr, ok := field.ValueExpression()
		if !ok {
			return ProviderScope{}, fmt.Errorf("wiring field %s has no value", fieldName)
		}
		if _, isNil := expr.(ast.NilLiteralNode); isNil {
			return ProviderScope{}, diagnosticf(fieldSpan, "providers-nil-wiring",
				"wiring value for %s must not be nil", fieldName)
		}
		valTypes, err := tc.inferExpressionType(expr)
		if err != nil {
			return ProviderScope{}, err
		}
		if len(valTypes) != 1 {
			return ProviderScope{}, fmt.Errorf("wiring field %s must have a single type", fieldName)
		}
		contractType, ok := eng.KnownRoots[fieldName]
		if !ok {
			return ProviderScope{}, diagnosticf(fieldSpan, "providers-unknown-key",
				"unknown wiring key %q", fieldName)
		}
		if !tc.wiringValueAssignable(valTypes[0], contractType) {
			return ProviderScope{}, diagnosticf(fieldSpan, "providers-wiring-type",
				"wiring field %s: expected type %s, got %s", fieldName, contractType.Ident, valTypes[0].Ident)
		}
		amb.Keys[fieldName] = contractType
		amb.Shadowed[fieldName] = true
	}
	return amb, nil
}

func shapeFieldSpan(_ ast.ShapeNode, _ string, field ast.ShapeFieldNode) ast.SourceSpan {
	if field.Node != nil {
		if expr, ok := field.Node.(ast.ExpressionNode); ok {
			if s := spanOfExpression(expr); s.IsSet() {
				return s
			}
		}
	}
	return ast.SourceSpan{}
}

func (tc *TypeChecker) ambientFromInferredBundle(types []ast.TypeNode, span ast.SourceSpan) (ProviderScope, error) {
	if len(types) != 1 {
		return ProviderScope{}, fmt.Errorf("with wiring must have a single type")
	}
	shape, ok := tc.shapeFieldsForType(types[0])
	if !ok {
		return ProviderScope{}, diagnosticf(span, "providers-wiring-shape",
			"with wiring expression must be a Providers shape")
	}
	eng := tc.providersEngine()
	amb := newProviderScope()
	for fieldName := range shape.Fields {
		if err := tc.validateWiringKey(fieldName, span); err != nil {
			return ProviderScope{}, err
		}
		contractType, ok := eng.KnownRoots[fieldName]
		if !ok {
			return ProviderScope{}, diagnosticf(span, "providers-unknown-key",
				"unknown wiring key %q", fieldName)
		}
		amb.Keys[fieldName] = contractType
		amb.Shadowed[fieldName] = true
	}
	return amb, nil
}

func (tc *TypeChecker) ambientFromWiringExpr(wiring ast.ExpressionNode, span ast.SourceSpan) (ProviderScope, error) {
	switch w := wiring.(type) {
	case ast.ShapeNode:
		return tc.providerScopeFromShape(w)
	default:
		types, err := tc.inferExpressionType(wiring)
		if err != nil {
			return ProviderScope{}, err
		}
		return tc.ambientFromInferredBundle(types, span)
	}
}

func (tc *TypeChecker) shapeFieldsForType(t ast.TypeNode) (ast.ShapeNode, bool) {
	if def, ok := tc.Defs[t.Ident]; ok {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if ae, ok := typeDefAssertionFromExpr(typeDef.Expr); ok && ae.Assertion != nil {
				fields := tc.resolveShapeFieldsFromAssertion(ae.Assertion)
				if len(fields) > 0 {
					return ast.ShapeNode{Fields: fields}, true
				}
			}
			if se, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				return se.Shape, true
			}
		}
	}
	return ast.ShapeNode{}, false
}

func (tc *TypeChecker) validateWiringKey(key string, span ast.SourceSpan) error {
	ident := ast.TypeIdent(key)
	if def, ok := tc.Defs[ident]; ok {
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if ae, ok := typeDefAssertionFromExpr(typeDef.Expr); ok && ae.Assertion != nil &&
				ae.Assertion.BaseType != nil && len(ae.Assertion.Constraints) == 0 {
				canonical := string(tc.providerRootIdent(ast.TypeNode{Ident: *ae.Assertion.BaseType}))
				if canonical != key {
					return diagnosticf(span, "providers-alias-key",
						"wiring key must be root contract ident %q, not alias %q", canonical, key)
				}
			}
		}
	}
	if _, ok := tc.providersEngine().KnownRoots[key]; ok {
		return nil
	}
	return diagnosticf(span, "providers-unknown-key", "unknown wiring key %q", key)
}

func (tc *TypeChecker) scopeSatisfiesSlot(slot ProviderSlot, scope map[string]ast.TypeNode) bool {
	if scope == nil {
		return false
	}
	contract, ok := scope[string(slot.RootIdent)]
	if !ok {
		return false
	}
	return tc.IsTypeCompatible(contract, slot.ContractType) || tc.IsTypeCompatible(slot.ContractType, contract)
}
