package ast

import "fmt"

// UseNode declares a Provider contract requirement in a function body.
// `use x: T` binds name x to contract T; `use T` is an anonymous use (root ident of T).
type UseNode struct {
	Ident        *Ident
	ContractType TypeNode
	Span         SourceSpan
}

func (u UseNode) Kind() NodeKind {
	return NodeKindUse
}

func (u UseNode) String() string {
	if u.Ident != nil {
		return fmt.Sprintf("Use(%s: %s)", u.Ident.ID, u.ContractType.Ident)
	}
	return fmt.Sprintf("Use(%s)", u.ContractType.Ident)
}
