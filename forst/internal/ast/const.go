package ast

// ConstSpec is one name in a const or const ( ... ) group.
type ConstSpec struct {
	Name  Ident
	Type  *TypeNode
	Value ExpressionNode // nil repeats the previous spec's expression in a group
}

// ConstGroupNode is a top-level const declaration (single spec or grouped).
type ConstGroupNode struct {
	Specs []ConstSpec
}

func (ConstGroupNode) Kind() NodeKind {
	return NodeKindConstGroup
}

func (n ConstGroupNode) String() string {
	if len(n.Specs) == 1 {
		s := "const " + string(n.Specs[0].Name.ID)
		if n.Specs[0].Type != nil {
			s += ": " + n.Specs[0].Type.String()
		}
		if n.Specs[0].Value != nil {
			s += " = " + n.Specs[0].Value.String()
		}
		return s
	}
	return "const (...)"
}
