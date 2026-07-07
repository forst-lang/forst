package typechecker

import (
	"fmt"
	"strconv"

	"forst/internal/ast"
	"forst/internal/hasher"
)

type scopePath string

// RebindScopes copies scope registrations from fromNodes (typechecked AST) onto
// matching nodes in toNodes (re-parsed AST) so RestoreScope works during transform.
func (tc *TypeChecker) RebindScopes(fromNodes, toNodes []ast.Node) {
	from := indexNodesByScopePath(fromNodes)
	to := indexNodesByScopePath(toNodes)
	for path, fromNode := range from {
		toNode, ok := to[path]
		if !ok || !tc.HasScopeForNode(fromNode) {
			continue
		}
		if hash, ok := tc.scopeStack.lookupRegisteredHash(fromNode); ok {
			tc.scopeStack.cacheNodeScopeHash(toNode, hash)
		}
	}
}

func (ss *ScopeStack) lookupRegisteredHash(node ast.Node) (NodeHash, bool) {
	if key, ok := hasher.NodeIdentityKey(node); ok {
		if hash, ok := ss.nodeScopeHash[key]; ok {
			return hash, true
		}
	}
	baseHash, err := ss.Hasher.HashScopeKey(node)
	if err != nil {
		return 0, false
	}
	if scope, ok := ss.scopes[baseHash]; ok && scopeNodesMatch(scope, node) {
		return baseHash, true
	}
	disHash, err := ss.Hasher.HashScopeKeyDisambiguated(node, baseHash)
	if err != nil {
		return 0, false
	}
	if _, ok := ss.scopes[disHash]; ok {
		return disHash, true
	}
	return 0, false
}

func indexNodesByScopePath(nodes []ast.Node) map[scopePath]ast.Node {
	out := make(map[scopePath]ast.Node)
	for i, n := range nodes {
		indexTopLevelNode(out, scopePath("top:"+strconv.Itoa(i)), n)
	}
	return out
}

func indexTopLevelNode(out map[scopePath]ast.Node, prefix scopePath, n ast.Node) {
	switch x := n.(type) {
	case ast.FunctionNode:
		fnPath := scopePath(fmt.Sprintf("%s:fn:%s", prefix, x.Ident.ID))
		out[fnPath] = n
		indexStmtPaths(out, fnPath, x.Body)
	case *ast.FunctionNode:
		if x != nil {
			indexTopLevelNode(out, prefix, *x)
		}
	case ast.TypeGuardNode:
		tgPath := scopePath(fmt.Sprintf("%s:tg:%s", prefix, x.Ident))
		out[tgPath] = n
		indexStmtPaths(out, tgPath, x.Body)
	case *ast.TypeGuardNode:
		if x != nil {
			indexTopLevelNode(out, prefix, *x)
		}
	}
}

func indexStmtPaths(out map[scopePath]ast.Node, prefix scopePath, stmts []ast.Node) {
	for i, stmt := range stmts {
		stmtPath := scopePath(fmt.Sprintf("%s/stmt:%d", prefix, i))
		indexScopedStmt(out, stmtPath, stmt)
	}
}

func indexScopedStmt(out map[scopePath]ast.Node, path scopePath, stmt ast.Node) {
	switch s := stmt.(type) {
	case *ast.IfNode:
		out[path+":if"] = stmt
		indexStmtPaths(out, path+"/if-then", s.Body)
		for j := range s.ElseIfs {
			eiPath := scopePath(fmt.Sprintf("%s/elseif:%d", path, j))
			out[eiPath+":elseif"] = &s.ElseIfs[j]
			indexStmtPaths(out, eiPath+"/body", s.ElseIfs[j].Body)
		}
		if s.Else != nil {
			out[path+":else"] = s.Else
			indexStmtPaths(out, path+"/else", s.Else.Body)
		}
	case ast.IfNode:
		indexScopedStmt(out, path, &s)
	case *ast.ForNode:
		out[path+":for"] = stmt
		indexStmtPaths(out, path+"/for", s.Body)
	case ast.ForNode:
		indexScopedStmt(out, path, &s)
	case ast.WithNode:
		out[path+":with"] = stmt
		indexStmtPaths(out, path+"/with", s.Body)
	case *ast.WithNode:
		if s != nil {
			indexScopedStmt(out, path, *s)
		}
	case ast.EnsureNode:
		if s.Block != nil {
			out[path+":ensure-block"] = s.Block
			indexStmtPaths(out, path+"/ensure", s.Block.Body)
		}
	case *ast.EnsureNode:
		if s != nil {
			indexScopedStmt(out, path, *s)
		}
	}
}
