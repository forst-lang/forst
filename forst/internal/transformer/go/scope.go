package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"
)

// restoreScope restores the scope for a given node
func (t *Transformer) restoreScope(node ast.Node) error {
	return t.TypeChecker.RestoreScope(node)
}

// currentScope returns the current scope from the type checker
func (t *Transformer) currentScope() *typechecker.Scope {
	return t.TypeChecker.CurrentScope()
}

// typeGuardScopeNode returns the ast.Node from nodes that owns guard's scope (same interface as typecheck).
func typeGuardScopeNode(nodes []ast.Node, guard ast.TypeGuardNode) ast.Node {
	for _, n := range nodes {
		if matchTypeGuardNode(n, guard) {
			return n
		}
	}
	return ast.Node(guard)
}

func matchTypeGuardNode(n ast.Node, guard ast.TypeGuardNode) bool {
	switch g := n.(type) {
	case ast.TypeGuardNode:
		return g.Ident == guard.Ident
	case *ast.TypeGuardNode:
		return g != nil && g.Ident == guard.Ident
	default:
		return false
	}
}

// resolveTypeGuardScopeNode finds the ast.Node whose scope was registered during typecheck.
// Module-level checking may register scopes on merged-package nodes while compile emits a single file.
func (t *Transformer) resolveTypeGuardScopeNode(entryNodes []ast.Node, guard ast.TypeGuardNode) ast.Node {
	return t.resolveScopeNode(entryNodes, func(n ast.Node) bool {
		return matchTypeGuardNode(n, guard)
	}, ast.Node(guard))
}

// resolveFunctionScopeNode finds the ast.Node whose scope was registered during typecheck.
func (t *Transformer) resolveFunctionScopeNode(entryNodes []ast.Node, fn ast.FunctionNode) ast.Node {
	return t.resolveScopeNode(entryNodes, func(n ast.Node) bool {
		if f, ok := n.(ast.FunctionNode); ok {
			return f.Ident.ID == fn.Ident.ID
		}
		return false
	}, ast.Node(fn))
}

func (t *Transformer) resolveScopeNode(entryNodes []ast.Node, match func(ast.Node) bool, fallback ast.Node) ast.Node {
	searchLists := [][]ast.Node{entryNodes}
	if t.moduleResult != nil {
		for _, merged := range t.moduleResult.PerPackageNodes {
			searchLists = append(searchLists, merged)
		}
	}
	for _, list := range searchLists {
		for _, n := range list {
			if match(n) && t.TypeChecker.HasScopeForNode(n) {
				return n
			}
		}
	}
	for _, list := range searchLists {
		for _, n := range list {
			if match(n) {
				return n
			}
		}
	}
	return fallback
}
