package ast

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// This file holds test-only helpers (fixture builders and SetupTestLogger). It is
// not part of the language AST model; production code may still call these from
// tests living in other packages.

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
	return IntLiteralNode{Value: value, Span: FakeSpan()}
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
func MakeShapeType(_ map[string]ShapeFieldNode) TypeNode {
	return TypeNode{
		Ident: TypeShape,
		TypeParams: []TypeNode{{
			Ident: TypeShape,
		}},
	}
}

// MakeStringLiteral creates a string literal
func MakeStringLiteral(value string) StringLiteralNode {
	return StringLiteralNode{Value: value, Span: FakeSpan()}
}

// MakeRuneLiteral creates a rune literal
func MakeRuneLiteral(r rune) RuneLiteralNode {
	return RuneLiteralNode{Value: int64(r), Span: FakeSpan()}
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
	return VariableNode{Ident: Ident{ID: Identifier(name), Span: FakeSpan()}}
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
		LValues: []ExpressionNode{VariableNode{
			Ident: Ident{ID: Identifier(varName), Span: FakeSpan()},
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
	sp := FakeSpan()
	return FunctionCallNode{
		Function:  Ident{ID: Identifier(functionName), Span: sp},
		Arguments: arguments,
		CallSpan:  sp,
	}
}

// MakePackage creates a package node
func MakePackage(name string, _ []Node) PackageNode {
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

// TestLoggerOptions contains options for setting up a test logger
type TestLoggerOptions struct {
	ForceLevel logrus.Level
}

// compilerTestLogsEnabled reports whether FORST_TEST_LOG requests live Debug logs.
// Values: 1|true|debug|all. Not tied to go test -v (that flag only affects test names).
func compilerTestLogsEnabled() bool {
	switch strings.ToLower(os.Getenv("FORST_TEST_LOG")) {
	case "1", "true", "debug", "all":
		return true
	default:
		return false
	}
}

func compilerTestLogsOnFail() bool {
	switch strings.ToLower(os.Getenv("FORST_TEST_LOG")) {
	case "fail", "onfail":
		return true
	default:
		return false
	}
}

// setupTestLogger is extracted so tests can cover the enabled branch without env.
// Quiet by default (Panic + Discard) so Debug/Trace call sites skip formatting/I/O.
func setupTestLogger(opts *TestLoggerOptions, enableDebug func() bool) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.PanicLevel)

	if enableDebug() {
		logger.SetOutput(os.Stderr)
		logger.SetLevel(logrus.DebugLevel)
	}

	if opts != nil {
		logger.SetLevel(opts.ForceLevel)
	}

	return logger
}

// SetupTestLogger creates a quiet test logger.
// Set FORST_TEST_LOG=1 for Debug on stderr. Prefer SetupTestLoggerFor to dump on failure.
func SetupTestLogger(opts *TestLoggerOptions) *logrus.Logger {
	return setupTestLogger(opts, compilerTestLogsEnabled)
}

// SetupTestLoggerFor is SetupTestLogger plus optional failure capture.
// With FORST_TEST_LOG=fail, buffers Debug logs and prints them via tb.Log if the test fails.
func SetupTestLoggerFor(tb testing.TB, opts *TestLoggerOptions) *logrus.Logger {
	tb.Helper()
	log := SetupTestLogger(opts)
	if !compilerTestLogsOnFail() {
		return log
	}
	var buf bytes.Buffer
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(&buf)
	tb.Cleanup(func() {
		if tb.Failed() && buf.Len() > 0 {
			tb.Logf("compiler logs (FORST_TEST_LOG=fail):\n%s", buf.String())
		}
	})
	return log
}
