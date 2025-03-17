package ast

// Package Node
type PackageNode struct {
	Ident Ident
}

// NodeType returns the type of this AST node
func (p PackageNode) NodeType() NodeType {
	return NodeTypePackage
}

func (p PackageNode) String() string {
	return p.Ident.String()
}
