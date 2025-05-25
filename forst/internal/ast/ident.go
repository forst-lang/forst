package ast

type Identifier string

// Ident represents an identifier
type Ident struct {
	Id Identifier
}

func (i *Ident) String() string {
	return string(i.Id)
}

func (i *Ident) Kind() NodeKind {
	return NodeKindIdentifier
}
