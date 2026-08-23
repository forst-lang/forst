package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

var allowedTypeParamConstraints = map[ast.TypeIdent]struct{}{
	ast.TypeIdent("any"):        {},
	ast.TypeIdent("comparable"): {},
}

func (tc *TypeChecker) validateFunctionTypeParamConstraints(sig FunctionSignature) error {
	for _, tp := range sig.TypeParams {
		if tp.Constraint == nil {
			continue
		}
		c := *tp.Constraint
		if c.Assertion != nil || len(c.TypeParams) > 0 {
			return fmt.Errorf("type parameter %s: constraint must be a simple identifier", tp.Name)
		}
		if _, ok := allowedTypeParamConstraints[c.Ident]; !ok {
			return fmt.Errorf("type parameter %s: unknown constraint %q (allowed: any, comparable)", tp.Name, c.Ident)
		}
	}
	return nil
}

func (tc *TypeChecker) checkTypeParamConstraints(sig FunctionSignature, bindings map[ast.TypeIdent]ast.TypeNode, span ast.SourceSpan) error {
	for _, tp := range sig.TypeParams {
		name := ast.TypeIdent(tp.Name)
		bound, ok := bindings[name]
		if !ok {
			continue
		}
		if tp.Constraint == nil {
			continue
		}
		constraint := tp.Constraint.Ident
		if constraint == ast.TypeIdent("any") {
			continue
		}
		if constraint == ast.TypeIdent("comparable") {
			if !isComparableForstType(bound) {
				return tc.genericDiag(span, fmt.Sprintf("type argument %s does not satisfy comparable constraint (type %s)", tp.Name, bound.String()))
			}
		}
	}
	return nil
}

func isComparableForstType(t ast.TypeNode) bool {
	if t.IsTypeParam() {
		return true
	}
	switch t.Ident {
	case ast.TypeInt, ast.TypeFloat, ast.TypeString, ast.TypeBool, ast.TypeBytes,
		ast.TypeComplex64, ast.TypeComplex128, ast.TypePointer:
		return true
	case ast.TypeArray:
		if t.IsSlice() {
			return false
		}
		return isComparableForstType(t.TypeParams[0])
	case ast.TypeMap:
		return false
	case ast.TypeFunc:
		return false
	case ast.TypeChannel:
		return false
	default:
		if t.IsUserDefined() || t.IsHashBased() {
			return true
		}
		return false
	}
}
