package ast

import (
	"fmt"
	"strings"
)

// ShapeNode represents a shape (object type) in the AST
type ShapeNode struct {
	ValueNode
	Fields map[string]ShapeFieldNode
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
	for name, field := range n.Fields {
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

func (n ShapeFieldNode) String() string {
	if n.IsMethod {
		return n.methodSignatureString()
	}
	if n.Shape != nil {
		return n.Shape.String()
	}
	if n.Assertion != nil {
		return n.Assertion.String()
	}
	if n.Type != nil {
		return n.Type.String()
	}
	return "?"
}

func (n ShapeFieldNode) methodSignatureString() string {
	var params []string
	for _, p := range n.MethodParams {
		params = append(params, p.String())
	}
	sig := fmt.Sprintf("(%s)", joinStrings(params, ", "))
	if len(n.MethodReturnTypes) > 0 {
		rets := make([]string, len(n.MethodReturnTypes))
		for i, rt := range n.MethodReturnTypes {
			rets[i] = rt.String()
		}
		sig += ": " + joinStrings(rets, ", ")
	}
	return sig
}

// IsMethodOnlyContract reports whether every field in the shape is a method signature (Usable contract).
func (s ShapeNode) IsMethodOnlyContract() bool {
	if len(s.Fields) == 0 {
		return false
	}
	for _, f := range s.Fields {
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
