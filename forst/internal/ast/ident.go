package ast

// Identifier is a unique identifier for names in the AST
type Identifier string

// Ident represents an identifier in the AST
type Ident struct {
	ID Identifier
}

// Kind returns the node kind for an identifier
func (i *Ident) Kind() NodeKind {
	return NodeKindIdentifier
}

// String returns the string representation of the identifier
func (i *Ident) String() string {
	return string(i.ID)
}
