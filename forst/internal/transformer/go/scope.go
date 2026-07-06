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
	if n, ok := t.TypeChecker.ScopeNodeForTypeGuard(guard.Ident); ok {
		return n
	}
	return t.resolveScopeNode(entryNodes, func(n ast.Node) bool {
		return matchTypeGuardNode(n, guard)
	}, ast.Node(guard))
}

// resolveFunctionScopeNode finds the ast.Node whose scope was registered during typecheck.
func (t *Transformer) resolveFunctionScopeNode(entryNodes []ast.Node, fn ast.FunctionNode) ast.Node {
	if n, ok := t.TypeChecker.ScopeNodeForFunction(fn.Ident.ID); ok {
		return n
	}
	return t.resolveScopeNode(entryNodes, func(n ast.Node) bool {
		if f, ok := n.(ast.FunctionNode); ok {
			return f.Ident.ID == fn.Ident.ID
		}
		return false
	}, ast.Node(fn))
}

func (t *Transformer) resolveWithScopeNode(entryNodes []ast.Node, with ast.WithNode) ast.Node {
	withStr := with.String()
	for _, list := range t.scopeSearchLists(entryNodes) {
		if n := findRegisteredWithScopeNode(list, withStr, t.TypeChecker); n != nil {
			return n
		}
	}
	return ast.Node(with)
}

func (t *Transformer) scopeSearchLists(entryNodes []ast.Node) [][]ast.Node {
	lists := [][]ast.Node{}
	if tcNodes := t.TypeChecker.TypecheckNodes(); len(tcNodes) > 0 {
		lists = append(lists, tcNodes)
	}
	lists = append(lists, entryNodes)
	if t.moduleResult != nil {
		for _, merged := range t.moduleResult.PerPackageNodes {
			lists = append(lists, merged)
		}
	}
	return lists
}

func findRegisteredWithScopeNode(nodes []ast.Node, withStr string, tc *typechecker.TypeChecker) ast.Node {
	for _, n := range nodes {
		switch x := n.(type) {
		case ast.FunctionNode:
			if found := findRegisteredWithInStmts(x.Body, withStr, tc); found != nil {
				return found
			}
		case *ast.FunctionNode:
			if x != nil {
				if found := findRegisteredWithInStmts(x.Body, withStr, tc); found != nil {
					return found
				}
			}
		case ast.TypeGuardNode:
			if found := findRegisteredWithInStmts(x.Body, withStr, tc); found != nil {
				return found
			}
		case *ast.TypeGuardNode:
			if x != nil {
				if found := findRegisteredWithInStmts(x.Body, withStr, tc); found != nil {
					return found
				}
			}
		}
	}
	return nil
}

func findRegisteredWithInStmts(stmts []ast.Node, withStr string, tc *typechecker.TypeChecker) ast.Node {
	for _, stmt := range stmts {
		if w, ok := stmt.(ast.WithNode); ok {
			if w.String() == withStr && tc.HasScopeForNode(stmt) {
				return stmt
			}
			if found := findRegisteredWithInStmts(w.Body, withStr, tc); found != nil {
				return found
			}
			continue
		}
		switch s := stmt.(type) {
		case ast.IfNode:
			if found := findRegisteredWithInIf(&s, withStr, tc); found != nil {
				return found
			}
		case *ast.IfNode:
			if s != nil {
				if found := findRegisteredWithInIf(s, withStr, tc); found != nil {
					return found
				}
			}
		case ast.ForNode:
			if found := findRegisteredWithInStmts(s.Body, withStr, tc); found != nil {
				return found
			}
		case *ast.ForNode:
			if s != nil {
				if found := findRegisteredWithInStmts(s.Body, withStr, tc); found != nil {
					return found
				}
			}
		case ast.EnsureNode:
			if s.Block != nil {
				if found := findRegisteredWithInStmts(s.Block.Body, withStr, tc); found != nil {
					return found
				}
			}
		case *ast.EnsureNode:
			if s != nil && s.Block != nil {
				if found := findRegisteredWithInStmts(s.Block.Body, withStr, tc); found != nil {
					return found
				}
			}
		}
	}
	return nil
}

func findRegisteredWithInIf(ifn *ast.IfNode, withStr string, tc *typechecker.TypeChecker) ast.Node {
	if found := findRegisteredWithInStmts(ifn.Body, withStr, tc); found != nil {
		return found
	}
	for i := range ifn.ElseIfs {
		if found := findRegisteredWithInStmts(ifn.ElseIfs[i].Body, withStr, tc); found != nil {
			return found
		}
	}
	if ifn.Else != nil {
		if found := findRegisteredWithInStmts(ifn.Else.Body, withStr, tc); found != nil {
			return found
		}
	}
	return nil
}

func (t *Transformer) resolveScopeNode(entryNodes []ast.Node, match func(ast.Node) bool, fallback ast.Node) ast.Node {
	searchLists := t.scopeSearchLists(entryNodes)
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
