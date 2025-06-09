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
}

// Kind returns the node kind for a shape
func (n ShapeNode) Kind() NodeKind {
	return NodeKindShape
}

func (n ShapeNode) isExpression() {}

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
	if n.Shape != nil {
		return n.Shape.String()
	}
	return n.Assertion.String()
}
