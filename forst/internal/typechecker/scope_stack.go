package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/hasher"

	"github.com/sirupsen/logrus"
)

// NodeHash is an alias for hasher.NodeHash
type NodeHash = hasher.NodeHash

// ScopeStack manages a stack of scopes during type checking
type ScopeStack struct {
	scopes        map[NodeHash]*Scope
	nodeScopeHash map[hasher.NodeIdentity]NodeHash
	current       *Scope
	Hasher        *hasher.StructuralHasher
	log           *logrus.Logger
}

// NewScopeStack creates a new stack with a global scope
func NewScopeStack(h *hasher.StructuralHasher, log *logrus.Logger) *ScopeStack {
	globalScope := NewScope(nil, nil, log)
	scopes := make(map[NodeHash]*Scope)
	nodeScopeHash := make(map[hasher.NodeIdentity]NodeHash)
	hash, err := h.HashNode(nil)
	if err != nil {
		log.WithError(err).Error("failed to hash nil node during NewScopeStack")
		return nil
	}
	scopes[hash] = globalScope
	globalScope.hash = hash

	return &ScopeStack{
		scopes:        scopes,
		nodeScopeHash: nodeScopeHash,
		current:       globalScope,
		Hasher:        h,
		log:           log,
	}
}

// pushScope creates and pushes a new scope for the given AST node
func (ss *ScopeStack) pushScope(node ast.Node) *Scope {
	hash, err := ss.Hasher.HashNode(node)
	if err != nil {
		ss.log.WithError(err).Error("failed to hash node during pushScope")
		return nil
	}
	scope := NewScope(ss.current, &node, ss.log)
	scope.hash = hash
	ss.current.Children = append(ss.current.Children, scope)
	ss.current = scope
	ss.scopes[hash] = scope
	if ss.nodeScopeHash == nil {
		ss.nodeScopeHash = make(map[hasher.NodeIdentity]NodeHash)
	}
	if key, ok := hasher.NodeIdentityKey(node); ok {
		ss.nodeScopeHash[key] = hash
	}
	return scope
}

// popScope returns to the parent scope if one exists
func (ss *ScopeStack) popScope() {
	if ss.current.Parent != nil {
		ss.current = ss.current.Parent
	}
}

// currentScope returns the current scope
func (ss *ScopeStack) currentScope() *Scope {
	return ss.current
}

// restoreScope restores a scope by its AST node
func (ss *ScopeStack) restoreScope(node ast.Node) error {
	scope, exists := ss.findScope(node)
	if !exists {
		return fmt.Errorf("scope not found for node %s", node.String())
	}
	ss.current = scope
	return nil
}

// findScope looks up a scope by its AST node
func (ss *ScopeStack) findScope(node ast.Node) (*Scope, bool) {
	if ss.nodeScopeHash == nil {
		ss.nodeScopeHash = make(map[hasher.NodeIdentity]NodeHash)
	}
	if key, ok := hasher.NodeIdentityKey(node); ok {
		if hash, ok := ss.nodeScopeHash[key]; ok {
			if scope, exists := ss.scopes[hash]; exists {
				return scope, true
			}
		}
	}
	hash, err := ss.Hasher.HashNode(node)
	if err != nil {
		ss.log.WithError(err).Error("failed to hash node during findScope")
		return nil, false
	}
	scope, exists := ss.scopes[hash]
	if exists {
		if key, ok := hasher.NodeIdentityKey(node); ok {
			ss.nodeScopeHash[key] = hash
		}
	}
	return scope, exists
}

// globalScope returns the root scope
func (ss *ScopeStack) globalScope() *Scope {
	scope := ss.current
	for scope.Parent != nil {
		scope = scope.Parent
	}
	return scope
}

// LookupVariableType looks up a variable's type in the current scope stack
func (ss *ScopeStack) LookupVariableType(name ast.Identifier) ([]ast.TypeNode, bool) {
	scope := ss.current
	for scope != nil {
		if types, exists := scope.LookupVariableType(name); exists {
			return types, true
		}
		scope = scope.Parent
	}
	return nil, false
}
