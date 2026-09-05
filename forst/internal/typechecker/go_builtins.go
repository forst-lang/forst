package typechecker

import (
	"forst/internal/ast"
)

// shapeExpectationMatches handles hash-typed shape literals assigned to a named parameter type that
// has no Defs entry but was bound during inferShapeType (see shapeExpectations on TypeChecker).
func (tc *TypeChecker) shapeExpectationMatches(actual ast.TypeNode, expected ast.TypeNode) bool {
	if tc.shapeExpectations == nil || expected.Ident == "" {
		return false
	}
	expShape, ok := tc.shapeExpectations[expected.Ident]
	if !ok {
		return false
	}
	actualDef, ok := tc.Defs[actual.Ident]
	if !ok {
		return false
	}
	actualShape, ok := tc.getShapeFromTypeDef(actualDef)
	if !ok {
		return false
	}
	return tc.shapesHaveSameStructure(*actualShape, expShape)
}

// siblingShapeTypeMatches reports whether actual is structurally compatible with a shape type
// defined in another Forst package (pkg.TypeName).
func (tc *TypeChecker) siblingShapeTypeMatches(actual ast.TypeNode, siblingType ast.TypeIdent) bool {
	td, ok := tc.resolveForstSiblingTypeDef(siblingType)
	if !ok {
		return false
	}
	expectedShape, ok := tc.getShapeFromTypeDef(td)
	if !ok {
		return false
	}
	actualDef, ok := tc.Defs[actual.Ident]
	if !ok {
		return false
	}
	actualShape, ok := tc.getShapeFromTypeDef(actualDef)
	if !ok {
		return false
	}
	return tc.shapesHaveSameStructure(*actualShape, *expectedShape)
}

func isScalarTypeIdent(id ast.TypeIdent) bool {
	switch id {
	case ast.TypeString, ast.TypeInt, ast.TypeFloat, ast.TypeBool:
		return true
	case ast.TypeIdent("byte"), ast.TypeIdent("rune"):
		return true
	default:
		return false
	}
}

func isIntFamilyIdent(id ast.TypeIdent) bool {
	switch id {
	case ast.TypeInt, ast.TypeIdent("byte"), ast.TypeIdent("rune"):
		return true
	default:
		return false
	}
}

func isByteSliceType(t ast.TypeNode) bool {
	if t.Ident == ast.TypeBytes {
		return true
	}
	if t.Ident != ast.TypeArray || len(t.TypeParams) != 1 {
		return false
	}
	return t.TypeParams[0].Ident == ast.TypeIdent("byte")
}

// getShapeFromTypeDef extracts the shape from a TypeDefNode if it is shape-backed (ordinary shape or error payload).
func (tc *TypeChecker) getShapeFromTypeDef(def ast.Node) (*ast.ShapeNode, bool) {
	if typeDef, ok := def.(ast.TypeDefNode); ok {
		return ast.PayloadShape(typeDef.Expr)
	}
	return nil, false
}

// shapesAreStructurallyIdentical returns true if two ShapeNodes have the same fields and types
func (tc *TypeChecker) shapesAreStructurallyIdentical(a, b ast.ShapeNode) bool {
	if len(a.Fields) != len(b.Fields) {
		return false
	}
	for name, fieldA := range a.Fields {
		fieldB, ok := b.Fields[name]
		if !ok {
			return false
		}
		if fieldA.Type != nil && fieldB.Type != nil {
			if fieldA.Type.Ident == "?" || fieldB.Type.Ident == "?" {
				continue
			}
			if fieldA.Type.Ident == fieldB.Type.Ident {
				continue
			}
			if tc.IsTypeCompatible(*fieldA.Type, *fieldB.Type) {
				continue
			}
			return false
		} else if fieldA.Shape != nil && fieldB.Shape != nil {
			if !tc.shapesAreStructurallyIdentical(*fieldA.Shape, *fieldB.Shape) {
				return false
			}
		} else if (fieldA.Type != nil) != (fieldB.Type != nil) || (fieldA.Shape != nil) != (fieldB.Shape != nil) {
			return false
		}
	}
	return true
}

func (tc *TypeChecker) inferBuiltinArgType(args []ast.ExpressionNode, i int, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) (ast.TypeNode, error) {
	if i < 0 || i >= len(args) {
		return ast.TypeNode{}, reportBodyf(callSpan, "builtin-call", "internal: missing argument %d", i+1)
	}
	sp := spanForCallArg(argSpans, i, args, callSpan)
	ts, err := tc.inferExpressionType(args[i])
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(ts) != 1 {
		return ast.TypeNode{}, reportBodyf(sp, "builtin-call", "argument %d must have a single type", i+1)
	}
	return ts[0], nil
}

// isMapIndexRValue is true when expr is a subscript whose target is a map type (rvalue read).
func (tc *TypeChecker) isMapIndexRValue(expr ast.IndexExpressionNode) bool {
	tts, err := tc.inferExpressionType(expr.Target)
	if err != nil || len(tts) != 1 {
		return false
	}
	return tts[0].Ident == ast.TypeMap && len(tts[0].TypeParams) >= 2
}
