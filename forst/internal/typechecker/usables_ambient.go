package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

func (tc *TypeChecker) currentMergedAmbient() map[string]ast.TypeNode {
	if len(tc.ambientStack) == 0 {
		return nil
	}
	merged := make(map[string]ast.TypeNode)
	for i := range tc.ambientStack {
		a := tc.ambientStack[i]
		for k, v := range a.keys {
			merged[k] = v
		}
	}
	return merged
}

func (tc *TypeChecker) mergeAmbient(outer Ambient, inner Ambient) Ambient {
	out := newAmbient()
	for k, v := range outer.keys {
		if inner.shadowed[k] {
			continue
		}
		out.keys[k] = v
	}
	for k, v := range inner.keys {
		out.keys[k] = v
	}
	for k := range inner.shadowed {
		out.shadowed[k] = true
	}
	return out
}

func (tc *TypeChecker) ambientFromShape(shape ast.ShapeNode) (Ambient, error) {
	amb := newAmbient()
	for fieldName, field := range shape.Fields {
		if field.IsMethod {
			continue
		}
		fieldSpan := shapeFieldSpan(shape, fieldName, field)
		if err := tc.validateWiringKey(fieldName, fieldSpan); err != nil {
			return Ambient{}, err
		}
		expr, ok := field.ValueExpression()
		if !ok {
			return Ambient{}, fmt.Errorf("wiring field %s has no value", fieldName)
		}
		if _, isNil := expr.(ast.NilLiteralNode); isNil {
			return Ambient{}, diagnosticf(fieldSpan, "usables-nil-wiring",
				"wiring value for %s must not be nil", fieldName)
		}
		valTypes, err := tc.inferExpressionType(expr)
		if err != nil {
			return Ambient{}, err
		}
		if len(valTypes) != 1 {
			return Ambient{}, fmt.Errorf("wiring field %s must have a single type", fieldName)
		}
		contractType, ok := tc.knownUsableRoots[fieldName]
		if !ok {
			return Ambient{}, diagnosticf(fieldSpan, "usables-unknown-key",
				"unknown wiring key %q", fieldName)
		}
		if !tc.wiringValueAssignable(valTypes[0], contractType) {
			return Ambient{}, diagnosticf(fieldSpan, "usables-wiring-type",
				"wiring field %s: expected type %s, got %s", fieldName, contractType.Ident, valTypes[0].Ident)
		}
		amb.keys[fieldName] = contractType
		amb.shadowed[fieldName] = true
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

func (tc *TypeChecker) ambientFromInferredBundle(types []ast.TypeNode, span ast.SourceSpan) (Ambient, error) {
	if len(types) != 1 {
		return Ambient{}, fmt.Errorf("with wiring must have a single type")
	}
	shape, ok := tc.shapeFieldsForType(types[0])
	if !ok {
		return Ambient{}, diagnosticf(span, "usables-wiring-shape",
			"with wiring expression must be a Usables shape")
	}
	amb := newAmbient()
	for fieldName := range shape.Fields {
		if err := tc.validateWiringKey(fieldName, span); err != nil {
			return Ambient{}, err
		}
		contractType, ok := tc.knownUsableRoots[fieldName]
		if !ok {
			return Ambient{}, diagnosticf(span, "usables-unknown-key",
				"unknown wiring key %q", fieldName)
		}
		amb.keys[fieldName] = contractType
		amb.shadowed[fieldName] = true
	}
	return amb, nil
}

func (tc *TypeChecker) ambientFromWiringExpr(wiring ast.ExpressionNode, span ast.SourceSpan) (Ambient, error) {
	switch w := wiring.(type) {
	case ast.ShapeNode:
		return tc.ambientFromShape(w)
	default:
		types, err := tc.inferExpressionType(wiring)
		if err != nil {
			return Ambient{}, err
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
				canonical := string(tc.usableRootIdent(ast.TypeNode{Ident: *ae.Assertion.BaseType}))
				if canonical != key {
					return diagnosticf(span, "usables-alias-key",
						"wiring key must be root contract ident %q, not alias %q", canonical, key)
				}
			}
		}
	}
	if _, ok := tc.knownUsableRoots[key]; ok {
		return nil
	}
	return diagnosticf(span, "usables-unknown-key", "unknown wiring key %q", key)
}

func (tc *TypeChecker) ambientSatisfiesSlot(slot UsableSlot, ambient map[string]ast.TypeNode) bool {
	if ambient == nil {
		return false
	}
	contract, ok := ambient[string(slot.RootIdent)]
	if !ok {
		return false
	}
	return tc.IsTypeCompatible(contract, slot.ContractType) || tc.IsTypeCompatible(slot.ContractType, contract)
}
