package typechecker

import "forst/internal/ast"

// Test utilities for creating AST nodes in tests - now using centralized ast package

// typeIdentPtr creates a pointer to a TypeIdent
func typeIdentPtr(s string) *ast.TypeIdent {
	ti := ast.TypeIdent(s)
	return &ti
}

// makeConstraint creates a ConstraintNode with the given name and shape argument
func makeConstraint(name string, shape *ast.ShapeNode) ast.ConstraintNode {
	return ast.MakeConstraint(name, shape)
}

// makeShape creates a ShapeNode with the given fields
func makeShape(fields map[string]ast.ShapeFieldNode) *ast.ShapeNode {
	shape := ast.MakeShape(fields)
	return &shape
}

// makeTypeField creates a ShapeFieldNode with a type
func makeTypeField(typeIdent ast.TypeIdent) ast.ShapeFieldNode {
	return ast.MakeShapeField(ast.TypeNode{Ident: typeIdent})
}

// makeShapeField creates a ShapeFieldNode with a nested shape
func makeShapeField(fields map[string]ast.ShapeFieldNode) ast.ShapeFieldNode {
	return ast.MakeNestedStructField(makeShape(fields))
}

// makeAssertionField creates a ShapeFieldNode with an assertion
func makeAssertionField(baseType ast.TypeIdent) ast.ShapeFieldNode {
	return ast.MakeAssertionField(baseType)
}

// makeValueNode creates a ValueNode from an integer literal
func makeValueNode(value int64) *ast.ValueNode {
	return ast.MakeValueNode(value)
}
