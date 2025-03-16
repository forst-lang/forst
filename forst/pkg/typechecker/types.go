package typechecker

import (
	"fmt"
	"strings"

	"forst/pkg/ast"
)

// FunctionSignature represents a function's type information
type FunctionSignature struct {
	Ident      ast.Ident
	Parameters []ParameterSignature
	ReturnType ast.TypeNode
}

type ParameterSignature struct {
	Ident ast.Ident
	Type  ast.TypeNode
}

func (p ParameterSignature) String() string {
	return fmt.Sprintf("%s: %s", p.Ident.Id, p.Type.String())
}

func (p ParameterSignature) Id() ast.Identifier {
	return p.Ident.Id
}

func (f FunctionSignature) String() string {
	paramStrings := make([]string, len(f.Parameters))
	for i, param := range f.Parameters {
		paramStrings[i] = param.String()
	}
	return fmt.Sprintf("%s(%s) -> %s", f.Ident.Id, strings.Join(paramStrings, ", "), f.ReturnType.String())
}

func (f FunctionSignature) Id() ast.Identifier {
	return f.Ident.Id
}
