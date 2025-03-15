package ast

// Ident represents an identifier
type Ident struct {
	Name string
}

func (i *Ident) String() string {
	return i.Name
}

func (i *Ident) NodeType() string {
	return "Ident"
}
