package typechecker

import "forst/pkg/ast"

// FunctionSignature represents a function's type information
type FunctionSignature struct {
	Ident          ast.Ident
	Parameters     []ParameterSignature
	ReturnType     ast.TypeNode
	IsTypeInferred bool
}

type ParameterSignature struct {
	Ident ast.Ident
	Type  ast.TypeNode
}
