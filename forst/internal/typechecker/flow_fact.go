package typechecker

import "forst/internal/ast"

// FlowTypeFact records flow-sensitive information for a binding at a program point.
// Layer 2 of the narrowing architecture: lattice ops (typeops) stay pure on InternalType;
// facts carry provenance for LSP (guards) and future FFI invalidation (§9.7) without mixing
// memory effects into Meet/Join.
type FlowTypeFact struct {
	Ident       ast.Identifier
	RefinedType ast.TypeNode
	// NarrowingTypeGuards names assertion predicates (built-in and user-defined) from if-branch / ensure narrowing.
	NarrowingTypeGuards []string
}

// MergeFlowFactsAtIfJoin applies JoinAfterIfMerge to each fact key shared with branch narrowing.
// outerByIdent maps each binding to its pre-if (enclosing scope) type.
func MergeFlowFactsAtIfJoin(
	tc *TypeChecker,
	outerByIdent map[ast.Identifier]ast.TypeNode,
	branchFacts []FlowTypeFact,
) map[ast.Identifier]ast.TypeNode {
	out := make(map[ast.Identifier]ast.TypeNode, len(outerByIdent))
	for id, outer := range outerByIdent {
		var refinements []ast.TypeNode
		for _, f := range branchFacts {
			if f.Ident == id {
				refinements = append(refinements, f.RefinedType)
			}
		}
		if len(refinements) == 0 {
			continue
		}
		out[id] = JoinAfterIfMerge(tc, outer, refinements)
	}
	return out
}
