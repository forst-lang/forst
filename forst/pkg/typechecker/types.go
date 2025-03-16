package typechecker

import (
	"fmt"
	"strings"

	"forst/pkg/ast"
)

// FunctionSignature represents a function's type information
type FunctionSignature struct {
	Ident       ast.Ident
	Parameters  []ParameterSignature
	ReturnTypes []ast.TypeNode
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
	returnStrings := make([]string, len(f.ReturnTypes))
	for i, ret := range f.ReturnTypes {
		returnStrings[i] = ret.String()
	}
	return fmt.Sprintf("%s(%s) -> %s", f.Ident.Id, strings.Join(paramStrings, ", "), strings.Join(returnStrings, ", "))
}

func (f FunctionSignature) Id() ast.Identifier {
	return f.Ident.Id
}
