package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func setupTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	return logger
}

func setupTypeChecker(log *logrus.Logger) *typechecker.TypeChecker {
	return typechecker.New(log, false)
}

func setupTransformer(tc *typechecker.TypeChecker, log *logrus.Logger) *Transformer {
	return New(tc, log)
}

// AST Builder primitives for unit tests

// makeTypeDef creates a type definition node
func makeTypeDef(name string, shape ast.ShapeNode) ast.TypeDefNode {
	return ast.TypeDefNode{
		Ident: ast.TypeIdent(name),
		Expr: ast.TypeDefShapeExpr{
			Shape: shape,
		},
	}
}

// makeShape creates a shape node with fields
func makeShape(fields map[string]ast.ShapeFieldNode) ast.ShapeNode {
	return ast.ShapeNode{
		Fields: fields,
	}
}

// makeShapeField creates a shape field with a type
func makeShapeField(fieldType ast.TypeNode) ast.ShapeFieldNode {
	return ast.ShapeFieldNode{
		Type: &fieldType,
	}
}

// makePointerType creates a pointer type
func makePointerType(baseType string) ast.TypeNode {
	return ast.TypeNode{
		Ident:      ast.TypePointer,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeIdent(baseType)}},
	}
}

// makeStringType creates a string type
func makeStringType() ast.TypeNode {
	return ast.TypeNode{Ident: ast.TypeString}
}

// makeStringLiteral creates a string literal value
func makeStringLiteral(value string) ast.StringLiteralNode {
	return ast.StringLiteralNode{Value: value}
}

// makeAddressOf creates an address-of expression (&expr)
func makeAddressOf(operand ast.ExpressionNode) ast.UnaryExpressionNode {
	return ast.UnaryExpressionNode{
		Operator: ast.TokenBitwiseAnd,
		Operand:  operand,
	}
}

// makeStructLiteral creates a struct literal with base type and fields
func makeStructLiteral(baseType string, fields map[string]ast.ShapeFieldNode) ast.ShapeNode {
	return ast.ShapeNode{
		BaseType: func() *ast.TypeIdent { t := ast.TypeIdent(baseType); return &t }(),
		Fields:   fields,
	}
}

// makeStructField creates a struct field with a node value
func makeStructField(node ast.Node) ast.ShapeFieldNode {
	return ast.ShapeFieldNode{
		Node: node,
	}
}

// makeStructFieldWithType creates a struct field with an explicit type
func makeStructFieldWithType(fieldType ast.TypeNode) ast.ShapeFieldNode {
	return ast.ShapeFieldNode{
		Type: &fieldType,
	}
}

// makeNestedStructField creates a struct field with a nested shape
func makeNestedStructField(shape *ast.ShapeNode) ast.ShapeFieldNode {
	return ast.ShapeFieldNode{
		Shape: shape,
	}
}

// makeShapeType creates a TypeNode representing a shape type
func makeShapeType(fields map[string]ast.ShapeFieldNode) ast.TypeNode {
	return ast.TypeNode{
		Ident:      ast.TypeShape,
		TypeParams: []ast.TypeNode{},
		// Forst shape types are structural, so we use a ShapeNode as a parameter
		// but for this test, we just use the Ident and fields
	}
}

// makeAssignment creates an assignment node for testing
func makeAssignment(varName string, varType ast.TypeNode, value ast.ExpressionNode) *ast.AssignmentNode {
	return &ast.AssignmentNode{
		LValues: []ast.VariableNode{{
			Ident:        ast.Ident{ID: ast.Identifier(varName)},
			ExplicitType: varType,
		}},
		RValues:       []ast.ExpressionNode{value},
		ExplicitTypes: []*ast.TypeNode{{Ident: varType.Ident}},
		IsShort:       false,
	}
}

// makeFunction creates a function node for testing
func makeFunction(name string, params []ast.ParamNode, body []ast.Node) ast.FunctionNode {
	return ast.FunctionNode{
		Ident:  ast.Ident{ID: ast.Identifier(name)},
		Params: params,
		Body:   body,
	}
}

// makeSimpleParam creates a simple parameter
func makeSimpleParam(name string, paramType ast.TypeNode) ast.SimpleParamNode {
	return ast.SimpleParamNode{
		Ident: ast.Ident{ID: ast.Identifier(name)},
		Type:  paramType,
	}
}

// makeFunctionCall creates a function call node
func makeFunctionCall(functionName string, arguments []ast.ExpressionNode) ast.FunctionCallNode {
	return ast.FunctionCallNode{
		Function:  ast.Ident{ID: ast.Identifier(functionName)},
		Arguments: arguments,
	}
}

// makePackage creates a package node with the given name and nodes
func makePackage(name string, nodes []ast.Node) ast.PackageNode {
	return ast.PackageNode{
		Ident: ast.Ident{ID: ast.Identifier(name)},
		// If PackageNode is later extended to hold children, add them here
	}
}

// makeTypeNode creates a type node with the given identifier
func makeTypeNode(typeName string) ast.TypeNode {
	return ast.TypeNode{Ident: ast.TypeIdent(typeName)}
}

// makeReferenceNode creates a variable node referencing the given identifier
func makeReferenceNode(name string) ast.VariableNode {
	return ast.VariableNode{
		Ident: ast.Ident{ID: ast.Identifier(name)},
	}
}
