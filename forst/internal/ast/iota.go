package ast

// IotaLiteralNode is the predeclared iota identifier in a const group.
type IotaLiteralNode struct{}

func (IotaLiteralNode) isExpression() {}

func (IotaLiteralNode) Kind() NodeKind {
	return NodeKindIotaLiteral
}

func (IotaLiteralNode) String() string {
	return "iota"
}
