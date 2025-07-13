package ast

import (
	"github.com/sirupsen/logrus"
)

// Testing utilities for creating AST nodes in tests

// TypeDefBuilder creates type definition nodes
type TypeDefBuilder struct{}

// NewTypeDefBuilder creates a new TypeDefBuilder
func NewTypeDefBuilder() *TypeDefBuilder {
	return &TypeDefBuilder{}
}

// MakeTypeDef creates a type definition node
func (b *TypeDefBuilder) MakeTypeDef(name string, shape ShapeNode) TypeDefNode {
	return TypeDefNode{
		Ident: TypeIdent(name),
		Expr: TypeDefShapeExpr{
			Shape: shape,
		},
	}
}

// ShapeBuilder creates shape nodes
type ShapeBuilder struct{}

// NewShapeBuilder creates a new ShapeBuilder
func NewShapeBuilder() *ShapeBuilder {
	return &ShapeBuilder{}
}

// MakeShape creates a shape node with fields
func (b *ShapeBuilder) MakeShape(fields map[string]ShapeFieldNode) ShapeNode {
	return ShapeNode{
		Fields: fields,
	}
}

// MakeStructLiteral creates a struct literal with base type and fields
func (b *ShapeBuilder) MakeStructLiteral(baseType string, fields map[string]ShapeFieldNode) ShapeNode {
	return ShapeNode{
		BaseType: func() *TypeIdent { t := TypeIdent(baseType); return &t }(),
		Fields:   fields,
	}
}

// FieldBuilder creates shape field nodes
type FieldBuilder struct{}

// NewFieldBuilder creates a new FieldBuilder
func NewFieldBuilder() *FieldBuilder {
	return &FieldBuilder{}
}

// MakeShapeField creates a shape field with a type
func (b *FieldBuilder) MakeShapeField(fieldType TypeNode) ShapeFieldNode {
	return ShapeFieldNode{
		Type: &fieldType,
	}
}

// MakeStructField creates a struct field with a node value
func (b *FieldBuilder) MakeStructField(node Node) ShapeFieldNode {
	return ShapeFieldNode{
		Node: node,
	}
}

// MakeStructFieldWithType creates a struct field with an explicit type
func (b *FieldBuilder) MakeStructFieldWithType(fieldType TypeNode) ShapeFieldNode {
	return ShapeFieldNode{
		Type: &fieldType,
	}
}

// MakeNestedStructField creates a struct field with a nested shape
func (b *FieldBuilder) MakeNestedStructField(shape *ShapeNode) ShapeFieldNode {
	return ShapeFieldNode{
		Shape: shape,
	}
}

// MakeAssertionField creates a ShapeFieldNode with an assertion
func (b *FieldBuilder) MakeAssertionField(baseType TypeIdent) ShapeFieldNode {
	return ShapeFieldNode{
		Assertion: &AssertionNode{
			BaseType: func() *TypeIdent { t := baseType; return &t }(),
		},
	}
}

// TypeBuilder creates type nodes
type TypeBuilder struct{}

// NewTypeBuilder creates a new TypeBuilder
func NewTypeBuilder() *TypeBuilder {
	return &TypeBuilder{}
}

// MakePointerType creates a pointer type
func (b *TypeBuilder) MakePointerType(baseType string) TypeNode {
	return TypeNode{
		Ident:      TypePointer,
		TypeParams: []TypeNode{{Ident: TypeIdent(baseType)}},
	}
}

// MakeStringType creates a string type
func (b *TypeBuilder) MakeStringType() TypeNode {
	return TypeNode{Ident: TypeString}
}

// MakeTypeNode creates a type node with the given identifier
func (b *TypeBuilder) MakeTypeNode(typeName string) TypeNode {
	return TypeNode{Ident: TypeIdent(typeName)}
}

// MakeShapeType creates a TypeNode representing a shape type
func (b *TypeBuilder) MakeShapeType(fields map[string]ShapeFieldNode) TypeNode {
	return TypeNode{
		Ident:      TypeShape,
		TypeParams: []TypeNode{},
		// Forst shape types are structural, so we use a ShapeNode as a parameter
		// but for this test, we just use the Ident and fields
	}
}

// ValueBuilder creates value nodes
type ValueBuilder struct{}

// NewValueBuilder creates a new ValueBuilder
func NewValueBuilder() *ValueBuilder {
	return &ValueBuilder{}
}

// MakeStringLiteral creates a string literal value
func (b *ValueBuilder) MakeStringLiteral(value string) StringLiteralNode {
	return StringLiteralNode{Value: value}
}

// MakeValueNode creates a ValueNode from an integer literal
func (b *ValueBuilder) MakeValueNode(value int64) *ValueNode {
	var v ValueNode = IntLiteralNode{Value: value}
	return &v
}

// ExpressionBuilder creates expression nodes
type ExpressionBuilder struct{}

// NewExpressionBuilder creates a new ExpressionBuilder
func NewExpressionBuilder() *ExpressionBuilder {
	return &ExpressionBuilder{}
}

// MakeAddressOf creates an address-of expression (&expr)
func (b *ExpressionBuilder) MakeAddressOf(operand ExpressionNode) UnaryExpressionNode {
	return UnaryExpressionNode{
		Operator: TokenBitwiseAnd,
		Operand:  operand,
	}
}

// MakeReferenceNode creates a variable node referencing the given identifier
func (b *ExpressionBuilder) MakeReferenceNode(name string) VariableNode {
	return VariableNode{
		Ident: Ident{ID: Identifier(name)},
	}
}

// FunctionBuilder creates function nodes
type FunctionBuilder struct{}

// NewFunctionBuilder creates a new FunctionBuilder
func NewFunctionBuilder() *FunctionBuilder {
	return &FunctionBuilder{}
}

// MakeFunction creates a function node for testing
func (b *FunctionBuilder) MakeFunction(name string, params []ParamNode, body []Node) FunctionNode {
	return FunctionNode{
		Ident:  Ident{ID: Identifier(name)},
		Params: params,
		Body:   body,
	}
}

// MakeSimpleParam creates a simple parameter
func (b *FunctionBuilder) MakeSimpleParam(name string, paramType TypeNode) SimpleParamNode {
	return SimpleParamNode{
		Ident: Ident{ID: Identifier(name)},
		Type:  paramType,
	}
}

// MakeFunctionCall creates a function call node
func (b *FunctionBuilder) MakeFunctionCall(functionName string, arguments []ExpressionNode) FunctionCallNode {
	return FunctionCallNode{
		Function:  Ident{ID: Identifier(functionName)},
		Arguments: arguments,
	}
}

// AssignmentBuilder creates assignment nodes
type AssignmentBuilder struct{}

// NewAssignmentBuilder creates a new AssignmentBuilder
func NewAssignmentBuilder() *AssignmentBuilder {
	return &AssignmentBuilder{}
}

// MakeAssignment creates an assignment node for testing
func (b *AssignmentBuilder) MakeAssignment(varName string, varType TypeNode, value ExpressionNode) *AssignmentNode {
	return &AssignmentNode{
		LValues: []VariableNode{{
			Ident:        Ident{ID: Identifier(varName)},
			ExplicitType: varType,
		}},
		RValues:       []ExpressionNode{value},
		ExplicitTypes: []*TypeNode{{Ident: varType.Ident}},
		IsShort:       false,
	}
}

// PackageBuilder creates package nodes
type PackageBuilder struct{}

// NewPackageBuilder creates a new PackageBuilder
func NewPackageBuilder() *PackageBuilder {
	return &PackageBuilder{}
}

// MakePackage creates a package node with the given name and nodes
func (b *PackageBuilder) MakePackage(name string, nodes []Node) PackageNode {
	return PackageNode{
		Ident: Ident{ID: Identifier(name)},
		// If PackageNode is later extended to hold children, add them here
	}
}

// ConstraintBuilder creates constraint nodes
type ConstraintBuilder struct{}

// NewConstraintBuilder creates a new ConstraintBuilder
func NewConstraintBuilder() *ConstraintBuilder {
	return &ConstraintBuilder{}
}

// MakeConstraint creates a ConstraintNode with the given name and shape argument
func (b *ConstraintBuilder) MakeConstraint(name string, shape *ShapeNode) ConstraintNode {
	return ConstraintNode{
		Name: name,
		Args: []ConstraintArgumentNode{
			{Shape: shape},
		},
	}
}

// Convenience functions for backward compatibility

// MakeTypeDef creates a type definition node (convenience function)
func MakeTypeDef(name string, shape ShapeNode) TypeDefNode {
	return NewTypeDefBuilder().MakeTypeDef(name, shape)
}

// MakeShape creates a shape node with fields (convenience function)
func MakeShape(fields map[string]ShapeFieldNode) ShapeNode {
	return NewShapeBuilder().MakeShape(fields)
}

// MakeShapeField creates a shape field with a type (convenience function)
func MakeShapeField(fieldType TypeNode) ShapeFieldNode {
	return NewFieldBuilder().MakeShapeField(fieldType)
}

// MakePointerType creates a pointer type (convenience function)
func MakePointerType(baseType string) TypeNode {
	return NewTypeBuilder().MakePointerType(baseType)
}

// MakeStringType creates a string type (convenience function)
func MakeStringType() TypeNode {
	return NewTypeBuilder().MakeStringType()
}

// MakeStringLiteral creates a string literal value (convenience function)
func MakeStringLiteral(value string) StringLiteralNode {
	return NewValueBuilder().MakeStringLiteral(value)
}

// MakeAddressOf creates an address-of expression (convenience function)
func MakeAddressOf(operand ExpressionNode) UnaryExpressionNode {
	return NewExpressionBuilder().MakeAddressOf(operand)
}

// MakeStructLiteral creates a struct literal with base type and fields (convenience function)
func MakeStructLiteral(baseType string, fields map[string]ShapeFieldNode) ShapeNode {
	return NewShapeBuilder().MakeStructLiteral(baseType, fields)
}

// MakeStructField creates a struct field with a node value (convenience function)
func MakeStructField(node Node) ShapeFieldNode {
	return NewFieldBuilder().MakeStructField(node)
}

// MakeStructFieldWithType creates a struct field with an explicit type (convenience function)
func MakeStructFieldWithType(fieldType TypeNode) ShapeFieldNode {
	return NewFieldBuilder().MakeStructFieldWithType(fieldType)
}

// MakeNestedStructField creates a struct field with a nested shape (convenience function)
func MakeNestedStructField(shape *ShapeNode) ShapeFieldNode {
	return NewFieldBuilder().MakeNestedStructField(shape)
}

// MakeShapeType creates a TypeNode representing a shape type (convenience function)
func MakeShapeType(fields map[string]ShapeFieldNode) TypeNode {
	return NewTypeBuilder().MakeShapeType(fields)
}

// MakeAssignment creates an assignment node for testing (convenience function)
func MakeAssignment(varName string, varType TypeNode, value ExpressionNode) *AssignmentNode {
	return NewAssignmentBuilder().MakeAssignment(varName, varType, value)
}

// MakeFunction creates a function node for testing (convenience function)
func MakeFunction(name string, params []ParamNode, body []Node) FunctionNode {
	return NewFunctionBuilder().MakeFunction(name, params, body)
}

// MakeSimpleParam creates a simple parameter (convenience function)
func MakeSimpleParam(name string, paramType TypeNode) SimpleParamNode {
	return NewFunctionBuilder().MakeSimpleParam(name, paramType)
}

// MakeFunctionCall creates a function call node (convenience function)
func MakeFunctionCall(functionName string, arguments []ExpressionNode) FunctionCallNode {
	return NewFunctionBuilder().MakeFunctionCall(functionName, arguments)
}

// MakePackage creates a package node with the given name and nodes (convenience function)
func MakePackage(name string, nodes []Node) PackageNode {
	return NewPackageBuilder().MakePackage(name, nodes)
}

// MakeTypeNode creates a type node with the given identifier (convenience function)
func MakeTypeNode(typeName string) TypeNode {
	return NewTypeBuilder().MakeTypeNode(typeName)
}

// MakeReferenceNode creates a variable node referencing the given identifier (convenience function)
func MakeReferenceNode(name string) VariableNode {
	return NewExpressionBuilder().MakeReferenceNode(name)
}

// MakeConstraint creates a ConstraintNode with the given name and shape argument (convenience function)
func MakeConstraint(name string, shape *ShapeNode) ConstraintNode {
	return NewConstraintBuilder().MakeConstraint(name, shape)
}

// MakeAssertionField creates a ShapeFieldNode with an assertion (convenience function)
func MakeAssertionField(baseType TypeIdent) ShapeFieldNode {
	return NewFieldBuilder().MakeAssertionField(baseType)
}

// MakeValueNode creates a ValueNode from an integer literal (convenience function)
func MakeValueNode(value int64) *ValueNode {
	return NewValueBuilder().MakeValueNode(value)
}

// SetupTestLogger creates a test logger with debug level
func SetupTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	return logger
}
