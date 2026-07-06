package typechecker

import "forst/internal/ast"

// ScopeNode is the ast.Node interface box registered with pushScope during typecheck.
type ScopeNode = ast.Node

type scopeOwners struct {
	typeGuards map[ast.Identifier]ScopeNode
	functions  map[ast.Identifier]ScopeNode
}

func newScopeOwners() scopeOwners {
	return scopeOwners{
		typeGuards: make(map[ast.Identifier]ScopeNode),
		functions:  make(map[ast.Identifier]ScopeNode),
	}
}

func (tc *TypeChecker) resetScopeOwners() {
	tc.scopeOwners = newScopeOwners()
}

func (tc *TypeChecker) registerTypeGuardScope(id ast.Identifier, node ScopeNode) {
	if tc.scopeOwners.typeGuards == nil {
		tc.scopeOwners.typeGuards = make(map[ast.Identifier]ScopeNode)
	}
	tc.scopeOwners.typeGuards[id] = node
}

func (tc *TypeChecker) registerFunctionScope(id ast.Identifier, node ScopeNode) {
	if tc.scopeOwners.functions == nil {
		tc.scopeOwners.functions = make(map[ast.Identifier]ScopeNode)
	}
	tc.scopeOwners.functions[id] = node
}

// ScopeNodeForTypeGuard returns the ScopeNode registered at collect for a type guard ident.
func (tc *TypeChecker) ScopeNodeForTypeGuard(id ast.Identifier) (ScopeNode, bool) {
	n, ok := tc.scopeOwners.typeGuards[id]
	return n, ok
}

// ScopeNodeForFunction returns the ScopeNode registered at collect for a function ident.
func (tc *TypeChecker) ScopeNodeForFunction(id ast.Identifier) (ScopeNode, bool) {
	n, ok := tc.scopeOwners.functions[id]
	return n, ok
}
