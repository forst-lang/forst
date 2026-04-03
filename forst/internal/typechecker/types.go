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

// ParameterSignature represents a function parameter's type information
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

// FormatTypeNodeDisplay formats a type for human-facing UI (e.g. LSP hover). It resolves hash-based
// structural types to a named alias in the alias chain when one exists, so hover matches what the
// user wrote when possible.
func (tc *TypeChecker) FormatTypeNodeDisplay(t ast.TypeNode) string {
	if tc == nil {
		return t.String()
	}
	d := tc.GetMostSpecificNonHashAlias(t)
	return d.String()
}

// FormatFunctionSignatureDisplay formats a function signature using FormatTypeNodeDisplay for every
// parameter and return type. Multiple return values (including value + Error) are shown as a
// parenthesized tuple, e.g. (String, Error).
func (tc *TypeChecker) FormatFunctionSignatureDisplay(sig FunctionSignature) string {
	paramStrings := make([]string, len(sig.Parameters))
	for i, param := range sig.Parameters {
		paramStrings[i] = fmt.Sprintf("%s: %s", param.Ident.ID, tc.FormatTypeNodeDisplay(param.Type))
	}
	retParts := make([]string, len(sig.ReturnTypes))
	for i, ret := range sig.ReturnTypes {
		retParts[i] = tc.FormatTypeNodeDisplay(ret)
	}
	var retStr string
	switch len(retParts) {
	case 0:
		retStr = "Void"
	case 1:
		retStr = retParts[0]
	default:
		retStr = "(" + strings.Join(retParts, ", ") + ")"
	}
	return fmt.Sprintf("%s(%s) -> %s", sig.Ident.ID, strings.Join(paramStrings, ", "), retStr)
}

// InferredTypesForVariableIdentifier returns inferred types for a bare identifier. Types come from
// tc.Types (VariableNode hashes include only the identifier) with a fallback to VariableTypes after
// declarations. Identifiers that are also function or type names should be handled by the caller
// first (e.g. prefer function signature hover for func f() { ... }'s name).
func (tc *TypeChecker) InferredTypesForVariableIdentifier(id ast.Identifier) ([]ast.TypeNode, bool) {
	if tc == nil {
		return nil, false
	}
	vn := ast.VariableNode{Ident: ast.Ident{ID: id}}
	hash, err := tc.Hasher.HashNode(vn)
	if err == nil {
		if t, ok := tc.Types[hash]; ok && len(t) > 0 {
			return t, true
		}
	}
	if t, ok := tc.VariableTypes[id]; ok && len(t) > 0 {
		return t, true
	}
	return nil, false
}
