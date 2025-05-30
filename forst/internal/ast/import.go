package ast

import "fmt"

// ImportNode represents a single Go-style import statement
type ImportNode struct {
	// The canonical import path (e.g., fmt, github.com/user/repo)
	Path string
	// The optional local name for the import (e.g., "f" in `import f "fmt"`)
	// If nil, no alias is specified
	Alias *Ident

	// Indicates if this is a blank import (import _ "pkg")
	// used only for its initialization side effects
	SideEffectOnly bool
}

// Kind returns the node kind for imports
func (i ImportNode) Kind() NodeKind {
	return NodeKindImport
}

func (i ImportNode) String() string {
	if i.Alias != nil {
		return fmt.Sprintf("Import(%s as %s)", i.Path, i.Alias.ID)
	}
	return fmt.Sprintf("Import(%s)", i.Path)
}

// ImportGroupNode represents a group of imports in parentheses
// like: import (
//
//	   "fmt"
//	   "os"
//	)
type ImportGroupNode struct {
	// Imports contains all the imports in this group
	Imports []ImportNode
}

// Kind returns the node kind for import groups
func (g ImportGroupNode) Kind() NodeKind {
	return NodeKindImportGroup
}

func (g ImportGroupNode) String() string {
	return fmt.Sprintf("ImportGroup(%v)", g.Imports)
}

// IsGrouped returns whether an import is part of a group
func (i ImportNode) IsGrouped() bool {
	// This can be determined by the parser and set on each ImportNode
	// For now, we'll return false as a placeholder
	return false
}
