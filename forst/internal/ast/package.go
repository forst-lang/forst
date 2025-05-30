package ast

import (
	"fmt"
)

// PackageNode represents a package declaration
type PackageNode struct {
	Ident Ident
}

// Kind returns the node kind for packages
func (p PackageNode) Kind() NodeKind {
	return NodeKindPackage
}

func (p PackageNode) String() string {
	return fmt.Sprintf("Package(%s)", p.Ident.ID)
}

// GetIdent returns the package identifier
func (p PackageNode) GetIdent() string {
	return string(p.Ident.ID)
}

// IsMainPackage returns whether this is the main package
func (p PackageNode) IsMainPackage() bool {
	return p.GetIdent() == "main"
}
