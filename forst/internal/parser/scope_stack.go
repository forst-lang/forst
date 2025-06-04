package parser

import "forst/internal/ast"

// ScopeStack manages a stack of scopes during type checking
type ScopeStack struct {
	scopes  []Scope
	current *Scope
}

// NewScopeStack creates a new stack with a global scope
func NewScopeStack() *ScopeStack {
	globalScope := Scope{
		Variables:    make(map[string]ast.TypeNode),
		FunctionName: "",
		IsTypeGuard:  false,
		IsGlobal:     true,
	}
	scopes := []Scope{globalScope}
	return &ScopeStack{
		scopes:  scopes,
		current: &globalScope,
	}
}

// PushScope creates and pushes a new scope for the given AST node
func (ss *ScopeStack) PushScope(scope *Scope) {
	if scope.IsGlobal {
		panic("Cannot push global scope")
	}
	ss.scopes = append(ss.scopes, *scope)
	ss.current = scope
}

func (ss *ScopeStack) PopScope() {
	if ss.current.IsGlobal {
		panic("Cannot pop global scope")
	}
	ss.scopes = ss.scopes[:len(ss.scopes)-1]
	ss.current = &ss.scopes[len(ss.scopes)-1]
}

func (ss *ScopeStack) CurrentScope() *Scope {
	return ss.current
}

func (ss *ScopeStack) GlobalScope() *Scope {
	return &ss.scopes[0]
}
