package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// validateReferencedTypesAfterCollect ensures every user-defined type name used in shape fields
// and function signatures was declared (after the collect pass). Catches typos like "Stringd"
// that parse as identifiers but are not valid built-ins or defined types.
func (tc *TypeChecker) validateReferencedTypesAfterCollect() error {
	if err := tc.validateApplicationGuardReceivers(); err != nil {
		return err
	}
	for _, def := range tc.Defs {
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			continue
		}
		if bin, ok := typeDef.Expr.(ast.TypeDefBinaryExpr); ok {
			if err := tc.validateTypeDefBinary(typeDef.Ident, bin); err != nil {
				return err
			}
		} else if err := tc.validateLiteralUnionExpr(typeDef.Ident, typeDef.Expr); err != nil {
			return err
		}
		payload, ok := ast.PayloadShape(typeDef.Expr)
		if !ok {
			continue
		}
		for fname, field := range payload.Fields {
			if field.Type != nil {
				ctx := fmt.Sprintf("type %s field %q", typeDef.Ident, fname)
				span := spanOfShapeField(field)
				if err := tc.validateTypeReference(*field.Type, span, ctx); err != nil {
					return err
				}
			}
		}
	}

	for _, sig := range tc.Functions {
		if err := tc.validateFunctionTypeParamConstraints(sig); err != nil {
			return err
		}
		fnName := string(sig.Ident.ID)
		for _, p := range sig.Parameters {
			ctx := fmt.Sprintf("function %s parameter %q", fnName, p.GetIdent())
			if err := tc.validateTypeReference(p.Type, p.Ident.Span, ctx); err != nil {
				return err
			}
		}
		for i, rt := range sig.ReturnTypes {
			ctx := fmt.Sprintf("function %s return[%d]", fnName, i)
			if err := tc.validateTypeReference(rt, sig.Ident.Span, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateApplicationGuardReceivers rejects `is (s String) Strong` style app guards.
// Builtin constraints (String.Min) remain; only user type guards on builtins are banned.
func (tc *TypeChecker) validateApplicationGuardReceivers() error {
	for _, def := range tc.Defs {
		var guard ast.TypeGuardNode
		switch g := def.(type) {
		case ast.TypeGuardNode:
			guard = g
		case *ast.TypeGuardNode:
			if g == nil {
				continue
			}
			guard = *g
		default:
			continue
		}
		subj := guard.Subject.GetType()
		span := ast.SourceSpan{}
		if sp, ok := guard.Subject.(ast.SimpleParamNode); ok {
			span = sp.Ident.Span
		}
		if tc.isBuiltinType(subj.Ident) {
			return reportf(span, "refinement-guard-receiver-mismatch",
				fmt.Sprintf("application guard `%s` cannot use builtin receiver", guard.Ident),
				fmt.Sprintf("Application guard `%s` cannot use builtin receiver `%s`.", guard.Ident, subj.Ident),
				fmt.Sprintf("declare `type Name = %s` and guard that type instead", subj.Ident))
		}
	}
	return nil
}

func (tc *TypeChecker) validateTypeReference(t ast.TypeNode, span ast.SourceSpan, ctx string) error {
	if t.Ident == ast.TypeImplicit {
		return nil
	}

	// Hand-built tests sometimes encode pointers as a single ident "*User" instead of TypePointer.
	if t.Ident != "" && strings.HasPrefix(string(t.Ident), "*") {
		inner := strings.TrimPrefix(string(t.Ident), "*")
		switch inner {
		case "String", "Int", "Float", "Bool", "Void":
			return nil
		default:
			return tc.validateTypeReference(ast.TypeNode{Ident: ast.TypeIdent(inner), TypeKind: ast.TypeKindUserDefined}, span, ctx)
		}
	}

	switch t.Ident {
	case ast.TypeArray:
		if len(t.TypeParams) == 0 {
			return nil
		}
		return tc.validateTypeReference(t.TypeParams[0], span, ctx+" element")
	case ast.TypeMap:
		if len(t.TypeParams) < 2 {
			return nil
		}
		if err := tc.validateTypeReference(t.TypeParams[0], span, ctx+" map key"); err != nil {
			return err
		}
		return tc.validateTypeReference(t.TypeParams[1], span, ctx+" map value")
	case ast.TypePointer:
		if len(t.TypeParams) == 0 {
			return nil
		}
		return tc.validateTypeReference(t.TypeParams[0], span, ctx+" pointer")
	case ast.TypeAssertion:
		if t.Assertion != nil && t.Assertion.BaseType != nil {
			return tc.validateTypeReference(ast.TypeNode{
				Ident:    *t.Assertion.BaseType,
				TypeKind: ast.TypeKindUserDefined,
			}, span, ctx+" assertion base")
		}
		return nil
	case ast.TypeShape:
		return nil
	case ast.TypeObject:
		return nil
	case ast.TypeResult:
		if len(t.TypeParams) >= 2 {
			if err := tc.validateTypeReference(t.TypeParams[0], span, ctx+" result success"); err != nil {
				return err
			}
			return tc.validateTypeReference(t.TypeParams[1], span, ctx+" result failure")
		}
		return nil
	case ast.TypeTuple:
		for i, p := range t.TypeParams {
			if err := tc.validateTypeReference(p, span, fmt.Sprintf("%s tuple[%d]", ctx, i)); err != nil {
				return err
			}
		}
		return nil
	case ast.TypeUnion, ast.TypeIntersection:
		for i, p := range t.TypeParams {
			if err := tc.validateTypeReference(p, span, fmt.Sprintf("%s %s[%d]", ctx, t.Ident, i)); err != nil {
				return err
			}
		}
		return nil
	default:
		if tc.isBuiltinType(t.Ident) {
			return nil
		}
		// Parser emits identifier "Error" for the built-in error type in some positions (e.g. tuple returns).
		if t.Ident == "Error" || t.Ident == ast.TypeError {
			return nil
		}
		// Synthetic / hand-built tests may use a non-AST type label (e.g. "Pointer(String)").
		if strings.ContainsAny(string(t.Ident), "()") {
			return nil
		}
		if IsGoBuiltinType(string(t.Ident)) {
			return nil
		}
		if strings.Contains(string(t.Ident), ".") {
			return nil
		}
		if t.TypeKind == ast.TypeKindHashBased {
			if _, ok := tc.Defs[t.Ident]; !ok {
				return reportf(span, "unknown-type",
					fmt.Sprintf("unknown structural type `%s`", t.Ident),
					fmt.Sprintf("%s: unknown structural type %q.", ctx, t.Ident),
					"declare the type or fix the spelling")
			}
			return nil
		}
		if t.IsTypeParam() {
			return nil
		}
		if strings.HasPrefix(string(t.Ident), "T_") {
			return nil
		}
		def, ok := tc.Defs[t.Ident]
		if !ok {
			return reportf(span, "unknown-type",
				fmt.Sprintf("unknown type `%s`", t.Ident),
				fmt.Sprintf("%s: unknown type %q.", ctx, t.Ident),
				"declare the type or fix the spelling")
		}
		if tn, ok := def.(ast.TypeNode); ok && (tn.IsTypeParam() || tn.TypeKind == ast.TypeKindUserDefined) {
			return nil
		}
		if _, ok := def.(ast.TypeDefNode); !ok {
			return reportf(span, "not-a-type-name",
				fmt.Sprintf("`%s` is not a type name", t.Ident),
				fmt.Sprintf("%s: %q is not a type name.", ctx, t.Ident),
				"use a declared type name or built-in type")
		}
		return nil
	}
}
