package typechecker

import "forst/internal/ast"

// Test utilities for creating AST nodes in tests

// typeIdentPtr creates a pointer to a TypeIdent
func typeIdentPtr(s string) *ast.TypeIdent {
	ti := ast.TypeIdent(s)
	return &ti
}

// makeConstraint creates a ConstraintNode with the given name and shape argument
func makeConstraint(name string, shape *ast.ShapeNode) ast.ConstraintNode {
	return ast.ConstraintNode{
		Name: name,
		Args: []ast.ConstraintArgumentNode{
			{Shape: shape},
		},
	}
}

// makeShape creates a ShapeNode with the given fields
func makeShape(fields map[string]ast.ShapeFieldNode) *ast.ShapeNode {
	return &ast.ShapeNode{Fields: fields}
}

// makeTypeField creates a ShapeFieldNode with a type
func makeTypeField(typeIdent ast.TypeIdent) ast.ShapeFieldNode {
	return ast.ShapeFieldNode{
		Type: &ast.TypeNode{Ident: typeIdent},
	}
}

// makeShapeField creates a ShapeFieldNode with a nested shape
func makeShapeField(fields map[string]ast.ShapeFieldNode) ast.ShapeFieldNode {
	return ast.ShapeFieldNode{
		Shape: makeShape(fields),
	}
}

// makeAssertionField creates a ShapeFieldNode with an assertion
func makeAssertionField(baseType ast.TypeIdent) ast.ShapeFieldNode {
	return ast.ShapeFieldNode{
		Assertion: &ast.AssertionNode{
			BaseType: typeIdentPtr(string(baseType)),
		},
	}
}

// makeValueNode creates a ValueNode from an integer literal
func makeValueNode(value int64) *ast.ValueNode {
	var v ast.ValueNode = ast.IntLiteralNode{Value: value}
	return &v
}
