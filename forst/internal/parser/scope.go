package parser

import "forst/internal/ast"

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
}

func (s *Scope) DefineVariable(name string, typeNode ast.TypeNode) {
	s.Variables[name] = typeNode
}
