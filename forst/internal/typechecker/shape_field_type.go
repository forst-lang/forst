package typechecker

import "forst/internal/ast"

// FieldTypeForNamedShape returns the declared field type for a named shape/alias type, if found.
func (tc *TypeChecker) FieldTypeForNamedShape(shapeTypeIdent ast.TypeIdent, fieldName string) (ast.TypeNode, bool) {
	if tc == nil {
		return ast.TypeNode{}, false
	}
	def, ok := tc.Defs[shapeTypeIdent]
	if !ok {
		return ast.TypeNode{}, false
	}
	td, ok := def.(ast.TypeDefNode)
	if !ok {
		return ast.TypeNode{}, false
	}
	shapePtr, ok := ast.PayloadShape(td.Expr)
	if !ok {
		return ast.TypeNode{}, false
	}
	f, ok := shapePtr.Fields[fieldName]
	if !ok || f.Type == nil {
		return ast.TypeNode{}, false
	}
	return *f.Type, true
}
