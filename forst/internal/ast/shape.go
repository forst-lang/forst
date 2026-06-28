package ast

import (
	"fmt"
	"strings"
)

// ShapeNode represents a shape (object type) in the AST
type ShapeNode struct {
	ValueNode
	Fields map[string]ShapeFieldNode
	// FieldOrder is source declaration order when parsed from Forst; empty means unspecified (printer may sort).
	FieldOrder []string
	// Optional explicit base type, e.g. "User" in User{ name: "John" }
	BaseType *TypeIdent
}

// ShapeFieldNode represents a field in a shape
type ShapeFieldNode struct {
	Node
	Assertion *AssertionNode
	Shape     *ShapeNode
	Type      *TypeNode
	// Method-only contract shapes use IsMethod with MethodParams / MethodReturnTypes.
	IsMethod           bool
	MethodParams       []ParamNode
	MethodReturnTypes  []TypeNode
}

// IsMethodField reports whether this shape entry is a method signature (not a data field).
func (f ShapeFieldNode) IsMethodField() bool {
	return f.IsMethod
}

// Kind returns the node kind for a shape
func (n ShapeNode) Kind() NodeKind {
	return NodeKindShape
}

func (n ShapeNode) isExpression() { _ = n }

func (n ShapeNode) String() string {
	var fields []string
	for _, name := range ShapeFieldNamesInOrder(n.Fields, n.FieldOrder) {
		field := n.Fields[name]
		var fieldStr string
		if field.Shape != nil {
			fieldStr = field.Shape.String()
		} else if field.Assertion != nil {
			fieldStr = field.Assertion.String()
		} else if field.Type != nil {
			fieldStr = field.Type.String()
		} else {
			fieldStr = "?"
		}
		fields = append(fields, fmt.Sprintf("%s: %s", name, fieldStr))
	}
	return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
}

func (f ShapeFieldNode) String() string {
	if f.IsMethod {
		return f.methodSignatureString()
	}
	if f.Shape != nil {
		return f.Shape.String()
	}
	if f.Assertion != nil {
		return f.Assertion.String()
	}
	if f.Type != nil {
		return f.Type.String()
	}
	return "?"
}

func (f ShapeFieldNode) methodSignatureString() string {
	var params []string
	for _, p := range f.MethodParams {
		params = append(params, p.String())
	}
	sig := fmt.Sprintf("(%s)", joinStrings(params, ", "))
	if len(f.MethodReturnTypes) > 0 {
		rets := make([]string, len(f.MethodReturnTypes))
		for i, rt := range f.MethodReturnTypes {
			rets[i] = rt.String()
		}
		sig += ": " + joinStrings(rets, ", ")
	}
	return sig
}

// IsMethodOnlyContract reports whether every field in the shape is a method signature (Provider contract).
func (n ShapeNode) IsMethodOnlyContract() bool {
	if len(n.Fields) == 0 {
		return false
	}
	for _, f := range n.Fields {
		if !f.IsMethod {
			return false
		}
	}
	return true
}

// ValueExpression returns the wiring or literal expression stored in a shape field, if any.
func (f ShapeFieldNode) ValueExpression() (ExpressionNode, bool) {
	if f.Shape != nil {
		return *f.Shape, true
	}
	if f.Node != nil {
		if expr, ok := f.Node.(ExpressionNode); ok {
			return expr, true
		}
	}
	return nil, false
}
