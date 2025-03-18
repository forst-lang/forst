package ast

import "fmt"

type TypeDefNode struct {
	Node
	Name string
}

func (t TypeDefNode) Kind() NodeKind {
	return NodeKindType
}

func (t TypeDefNode) String() string {
	return fmt.Sprintf("TypeDefNode(%s)", t.Name)
}
