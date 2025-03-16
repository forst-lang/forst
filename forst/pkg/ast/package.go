package ast

// Package Node
type PackageNode struct {
	Ident Ident
}

// NodeType returns the type of this AST node
func (p PackageNode) NodeType() string {
	return "Package"
}

func (p PackageNode) String() string {
	return p.Ident.String()
}
