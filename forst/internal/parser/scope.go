package parser

import (
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// Scope represents a scope in the parsed program
type Scope struct {
	// All variables defined in the scope
	Variables map[string]ast.TypeNode
	// The function currently being parsed
	FunctionName string
	// Whether this scope is a type guard
	IsTypeGuard bool
	// Whether this scope is the global scope
	IsGlobal bool
	// Logger for the scope
	log *logrus.Logger
}

// NewScope creates a new scope
func NewScope(functionName string, isGlobal bool, isTypeGuard bool, log *logrus.Logger) *Scope {
	return &Scope{
		Variables:    make(map[string]ast.TypeNode),
		FunctionName: functionName,
		IsTypeGuard:  isTypeGuard,
		IsGlobal:     isGlobal,
		log:          log,
	}
}

func (s *Scope) DefineVariable(name string, typeNode ast.TypeNode) {
	s.Variables[name] = typeNode
}
