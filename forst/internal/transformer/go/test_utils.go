package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func setupTestLogger() *logrus.Logger {
	return ast.SetupTestLogger()
}

func setupTypeChecker(log *logrus.Logger) *typechecker.TypeChecker {
	return typechecker.New(log, false)
}

func setupTransformer(tc *typechecker.TypeChecker, log *logrus.Logger) *Transformer {
	return New(tc, log)
}

// AST Builder primitives for unit tests - now using centralized ast package

// makeTypeDef creates a type definition node
func makeTypeDef(name string, shape ast.ShapeNode) ast.TypeDefNode {
	return ast.MakeTypeDef(name, shape)
}

// makeShape creates a shape node with fields
func makeShape(fields map[string]ast.ShapeFieldNode) ast.ShapeNode {
	return ast.MakeShape(fields)
}

// makeShapeField creates a shape field with a type
func makeShapeField(fieldType ast.TypeNode) ast.ShapeFieldNode {
	return ast.MakeShapeField(fieldType)
}

// makePointerType creates a pointer type
func makePointerType(baseType string) ast.TypeNode {
	return ast.MakePointerType(baseType)
}

// makeStringType creates a string type
func makeStringType() ast.TypeNode {
	return ast.MakeStringType()
}

// makeStringLiteral creates a string literal value
func makeStringLiteral(value string) ast.StringLiteralNode {
	return ast.MakeStringLiteral(value)
}

// makeAddressOf creates an address-of expression (&expr)
func makeAddressOf(operand ast.ExpressionNode) ast.UnaryExpressionNode {
	return ast.MakeAddressOf(operand)
}

// makeStructLiteral creates a struct literal with base type and fields
func makeStructLiteral(baseType string, fields map[string]ast.ShapeFieldNode) ast.ShapeNode {
	return ast.MakeStructLiteral(baseType, fields)
}

// makeStructField creates a struct field with a node value
func makeStructField(node ast.Node) ast.ShapeFieldNode {
	return ast.MakeStructField(node)
}

// makeStructFieldWithType creates a struct field with an explicit type
func makeStructFieldWithType(fieldType ast.TypeNode) ast.ShapeFieldNode {
	return ast.MakeStructFieldWithType(fieldType)
}

// makeNestedStructField creates a struct field with a nested shape
func makeNestedStructField(shape *ast.ShapeNode) ast.ShapeFieldNode {
	return ast.MakeNestedStructField(shape)
}

// makeShapeType creates a TypeNode representing a shape type
func makeShapeType(fields map[string]ast.ShapeFieldNode) ast.TypeNode {
	return ast.MakeShapeType(fields)
}

// makeAssignment creates an assignment node for testing
func makeAssignment(varName string, varType ast.TypeNode, value ast.ExpressionNode) *ast.AssignmentNode {
	return ast.MakeAssignment(varName, varType, value)
}

// makeFunction creates a function node for testing
func makeFunction(name string, params []ast.ParamNode, body []ast.Node) ast.FunctionNode {
	return ast.MakeFunction(name, params, body)
}

// makeSimpleParam creates a simple parameter
func makeSimpleParam(name string, paramType ast.TypeNode) ast.SimpleParamNode {
	return ast.MakeSimpleParam(name, paramType)
}

// makeFunctionCall creates a function call node
func makeFunctionCall(functionName string, arguments []ast.ExpressionNode) ast.FunctionCallNode {
	return ast.MakeFunctionCall(functionName, arguments)
}

// makePackage creates a package node with the given name and nodes
func makePackage(name string, nodes []ast.Node) ast.PackageNode {
	return ast.MakePackage(name, nodes)
}

// makeTypeNode creates a type node with the given identifier
func makeTypeNode(typeName string) ast.TypeNode {
	return ast.MakeTypeNode(typeName)
}

// makeReferenceNode creates a variable node referencing the given identifier
func makeReferenceNode(name string) ast.VariableNode {
	return ast.MakeReferenceNode(name)
}
