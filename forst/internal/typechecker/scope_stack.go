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
	hash, err := ss.resolveScopeHash(node)
	if err != nil {
		ss.log.WithError(err).Error("failed to hash node during pushScope")
		return nil
	}
	if existing, ok := ss.scopes[hash]; ok && scopeNodesMatch(existing, node) && existing.Parent == ss.current {
		ss.current = existing
		ss.cacheNodeScopeHash(node, hash)
		return existing
	}
	scope := NewScope(ss.current, &node, ss.log)
	scope.hash = hash
	ss.current.Children = append(ss.current.Children, scope)
	ss.current = scope
	ss.scopes[hash] = scope
	ss.cacheNodeScopeHash(node, hash)
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
	if node == nil {
		ss.current = ss.globalScope()
		return nil
	}
	scope, exists := ss.findScope(node)
	if !exists {
		desc := "<nil>"
		if node != nil {
			desc = node.String()
		}
		return fmt.Errorf("scope not found for node %s", desc)
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
	baseHash, err := ss.Hasher.HashScopeKey(node)
	if err != nil {
		ss.log.WithError(err).Error("failed to hash node during findScope")
		return nil, false
	}
	if scope, ok := ss.scopes[baseHash]; ok {
		if scopeNodesMatch(scope, node) {
			ss.cacheNodeScopeHash(node, baseHash)
			return scope, true
		}
		disHash, err := ss.Hasher.HashScopeKeyDisambiguated(node, baseHash)
		if err != nil {
			ss.log.WithError(err).Error("failed to disambiguate scope hash during findScope")
			return nil, false
		}
		if disScope, ok := ss.scopes[disHash]; ok {
			ss.cacheNodeScopeHash(node, disHash)
			return disScope, true
		}
		return nil, false
	}
	disHash, err := ss.Hasher.HashScopeKeyDisambiguated(node, baseHash)
	if err != nil {
		ss.log.WithError(err).Error("failed to disambiguate scope hash during findScope")
		return nil, false
	}
	if scope, ok := ss.scopes[disHash]; ok {
		ss.cacheNodeScopeHash(node, disHash)
		return scope, true
	}
	return nil, false
}

func (ss *ScopeStack) resolveScopeHash(node ast.Node) (NodeHash, error) {
	baseHash, err := ss.Hasher.HashScopeKey(node)
	if err != nil {
		return 0, err
	}
	if existing, ok := ss.scopes[baseHash]; ok && !scopeNodesMatch(existing, node) {
		return ss.Hasher.HashScopeKeyDisambiguated(node, baseHash)
	}
	return baseHash, nil
}

func (ss *ScopeStack) cacheNodeScopeHash(node ast.Node, hash NodeHash) {
	if ss.nodeScopeHash == nil {
		ss.nodeScopeHash = make(map[hasher.NodeIdentity]NodeHash)
	}
	if key, ok := hasher.NodeIdentityKey(node); ok {
		ss.nodeScopeHash[key] = hash
	}
}

func scopeNodesMatch(scope *Scope, node ast.Node) bool {
	if scope == nil || scope.Node == nil {
		return false
	}
	idNode, okNode := hasher.NodeIdentityKey(node)
	idScope, okScope := hasher.NodeIdentityKey(*scope.Node)
	if okNode && okScope {
		return idNode == idScope
	}
	return false
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
