package ast

// Package Node
type PackageNode struct {
	Value string
}

// NodeType returns the type of this AST node
func (p PackageNode) NodeType() string {
	return "Package"
} 
