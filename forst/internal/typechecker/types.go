package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"
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

// String returns a string representation of the parameter signature
func (p ParameterSignature) String() string {
	return fmt.Sprintf("%s: %s", p.Ident.ID, p.Type.String())
}

// GetIdent returns the parameter identifier as a string
func (p ParameterSignature) GetIdent() string {
	return string(p.Ident.ID)
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
	return fmt.Sprintf("%s(%s) -> %s", f.Ident.ID, strings.Join(paramStrings, ", "), strings.Join(returnStrings, ", "))
}

// GetIdent returns the function identifier as a string
func (f FunctionSignature) GetIdent() string {
	return string(f.Ident.ID)
}
