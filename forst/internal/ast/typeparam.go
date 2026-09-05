package ast

// TypeParamDecl is a declared type parameter on a generic function or type.
type TypeParamDecl struct {
	Name       Identifier
	Constraint *TypeNode // optional; nil means any
}
