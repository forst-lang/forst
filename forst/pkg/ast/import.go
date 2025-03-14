package ast

// ImportNode represents a single Go-style import statement
type ImportNode struct {
	// Path is the import path (e.g., "fmt", "github.com/user/repo")
	Path string
	
	// Alias is the optional local name for the import (e.g., "f" in `import f "fmt"`)
	// If nil, no alias is specified
	Alias *string
	
	// SideEffectOnly indicates if this is a blank import (import _ "pkg")
	// used only for its initialization side effects
	SideEffectOnly bool
}

// NodeType returns the type of this AST node
func (i ImportNode) NodeType() string {
	return "Import"
}

// ImportGroupNode represents a group of imports in parentheses
// like: import (
//          "fmt"
//          "os"
//       )
type ImportGroupNode struct {
	// Imports contains all the imports in this group
	Imports []ImportNode
}

// NodeType returns the type of this AST node
func (g ImportGroupNode) NodeType() string {
	return "ImportGroup"
}

// IsGrouped returns whether an import is part of a group
func (i ImportNode) IsGrouped() bool {
	// This can be determined by the parser and set on each ImportNode
	// For now, we'll return false as a placeholder
	return false
} 