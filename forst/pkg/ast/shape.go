package ast

import (
	"fmt"
	"strings"
)

type ShapeNode struct {
	Node
	Fields map[string]ShapeFieldNode
}

type ShapeFieldNode struct {
	Node
	Assertion *AssertionNode
	Shape     *ShapeNode
}

func (n ShapeNode) Kind() NodeKind {
	return NodeKindShape
}

func (n ShapeNode) String() string {
	var fields []string
	for name, field := range n.Fields {
		var fieldStr string
		if field.Shape != nil {
			fieldStr = field.Shape.String()
		} else if field.Assertion != nil {
			fieldStr = field.Assertion.String()
		}
		fields = append(fields, fmt.Sprintf("%s: %s", name, fieldStr))
	}
	return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
}
