package ast

import (
	"github.com/sirupsen/logrus"
)

// Testing utilities for creating AST nodes in tests

// MakeTypeDef creates a type definition node
func MakeTypeDef(name string, shape ShapeNode) TypeDefNode {
	return TypeDefNode{
		Ident: TypeIdent(name),
		Expr: TypeDefShapeExpr{
			Shape: shape,
		},
	}
}

// MakeShape creates a shape node with fields
func MakeShape(fields map[string]ShapeFieldNode) ShapeNode {
	return ShapeNode{
		Fields: fields,
	}
}

// MakeShapePtr creates a pointer to a shape node
func MakeShapePtr(fields map[string]ShapeFieldNode) *ShapeNode {
	shape := MakeShape(fields)
	return &shape
}

// MakeTypeField creates a shape field with a type
func MakeTypeField(typeIdent TypeIdent) ShapeFieldNode {
	return ShapeFieldNode{
		Type: &TypeNode{Ident: typeIdent},
	}
}

// MakeShapeField creates a shape field with nested shape
func MakeShapeField(fields map[string]ShapeFieldNode) ShapeFieldNode {
	return ShapeFieldNode{
		Shape: &ShapeNode{Fields: fields},
	}
}

// MakeAssertionField creates a shape field with assertion
func MakeAssertionField(baseType TypeIdent) ShapeFieldNode {
	return ShapeFieldNode{
		Assertion: &AssertionNode{
			BaseType: &baseType,
		},
	}
}

// MakeValueNode creates a value node
func MakeValueNode(value int64) ValueNode {
	return IntLiteralNode{Value: value}
}

// MakePointerType creates a pointer type
func MakePointerType(baseType string) TypeNode {
	return TypeNode{
		Ident: TypeIdent("*" + baseType),
	}
}

// MakeStringType creates a string type
func MakeStringType() TypeNode {
	return TypeNode{Ident: TypeString}
}

// MakeTypeNode creates a type node
func MakeTypeNode(typeName string) TypeNode {
	return TypeNode{Ident: TypeIdent(typeName)}
}

// MakeShapeType creates a shape type
func MakeShapeType(fields map[string]ShapeFieldNode) TypeNode {
	return TypeNode{
		Ident: TypeShape,
		TypeParams: []TypeNode{{
			Ident: TypeShape,
		}},
	}
}

// MakeStringLiteral creates a string literal
func MakeStringLiteral(value string) StringLiteralNode {
	return StringLiteralNode{Value: value}
}

// MakeAddressOf creates an address-of expression
func MakeAddressOf(operand ExpressionNode) UnaryExpressionNode {
	return UnaryExpressionNode{
		Operator: "&",
		Operand:  operand,
	}
}

// MakeReferenceNode creates a variable reference
func MakeReferenceNode(name string) VariableNode {
	return VariableNode{Ident: Ident{ID: Identifier(name)}}
}

// MakeStructLiteral creates a struct literal
func MakeStructLiteral(baseType string, fields map[string]ShapeFieldNode) ShapeNode {
	baseTypeIdent := TypeIdent(baseType)
	return ShapeNode{
		BaseType: &baseTypeIdent,
		Fields:   fields,
	}
}

// MakeStructField creates a struct field
func MakeStructField(node Node) ShapeFieldNode {
	return ShapeFieldNode{Node: node}
}

// MakeStructFieldWithType creates a struct field with type
func MakeStructFieldWithType(fieldType TypeNode) ShapeFieldNode {
	return ShapeFieldNode{Type: &fieldType}
}

// MakeNestedStructField creates a nested struct field
func MakeNestedStructField(shape *ShapeNode) ShapeFieldNode {
	return ShapeFieldNode{Shape: shape}
}

// MakeAssignment creates an assignment node
func MakeAssignment(varName string, varType TypeNode, value ExpressionNode) *AssignmentNode {
	return &AssignmentNode{
		LValues: []VariableNode{{
			Ident: Ident{ID: Identifier(varName)},
		}},
		RValues:       []ExpressionNode{value},
		ExplicitTypes: []*TypeNode{&varType},
		IsShort:       false,
	}
}

// MakeFunction creates a function node
func MakeFunction(name string, params []ParamNode, body []Node) FunctionNode {
	return FunctionNode{
		Ident:  Ident{ID: Identifier(name)},
		Params: params,
		Body:   body,
	}
}

// MakeSimpleParam creates a simple parameter
func MakeSimpleParam(name string, paramType TypeNode) SimpleParamNode {
	return SimpleParamNode{
		Ident: Ident{ID: Identifier(name)},
		Type:  paramType,
	}
}

// MakeFunctionCall creates a function call
func MakeFunctionCall(functionName string, arguments []ExpressionNode) FunctionCallNode {
	return FunctionCallNode{
		Function:  Ident{ID: Identifier(functionName)},
		Arguments: arguments,
	}
}

// MakePackage creates a package node
func MakePackage(name string, nodes []Node) PackageNode {
	return PackageNode{
		Ident: Ident{ID: Identifier(name)},
	}
}

// MakeConstraint creates a constraint node
func MakeConstraint(name string, shape *ShapeNode) ConstraintNode {
	return ConstraintNode{
		Name: name,
		Args: []ConstraintArgumentNode{
			{Shape: shape},
		},
	}
}

// SetupTestLogger creates a test logger
func SetupTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}
