package parser

import (
	"github.com/sirupsen/logrus"
)

// ScopeStack manages a stack of scopes during type checking
type ScopeStack struct {
	scopes  []*Scope
	current *Scope
	log     *logrus.Logger
}

// NewScopeStack creates a new stack with a global scope
func NewScopeStack(log *logrus.Logger) *ScopeStack {
	globalScope := NewScope("", true, false, log)
	scopes := []*Scope{globalScope}
	return &ScopeStack{
		scopes:  scopes,
		current: globalScope,
		log:     log,
	}
}

// PushScope creates and pushes a new scope for the given AST node
func (ss *ScopeStack) PushScope(scope *Scope) {
	if scope.IsGlobal {
		ss.log.Fatalf("Cannot push global scope")
	}
	ss.scopes = append(ss.scopes, scope)
	ss.current = scope
}

func (ss *ScopeStack) PopScope() {
	if ss.current.IsGlobal {
		ss.log.Fatalf("Cannot pop global scope")
	}
	ss.scopes = ss.scopes[:len(ss.scopes)-1]
	ss.current = ss.scopes[len(ss.scopes)-1]
}

func (ss *ScopeStack) CurrentScope() *Scope {
	return ss.current
}

func (ss *ScopeStack) GlobalScope() *Scope {
	return ss.scopes[0]
}
