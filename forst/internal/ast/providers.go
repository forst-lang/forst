package ast

import (
	"strings"
	"unicode"
)

// IsTestingTParamType reports whether t is *testing.T (or a pointer to a qualified testing.T).
func IsTestingTParamType(t TypeNode) bool {
	if t.Ident == TypePointer && len(t.TypeParams) == 1 {
		inner := string(t.TypeParams[0].Ident)
		return strings.HasSuffix(inner, "testing.T") || inner == "testing.T"
	}
	return false
}

// IsProvidersWiringRoot reports whether fnIdent is a wiring root (main or Test* with *testing.T).
// When paramTypes is empty for a Test* ident, the function is treated as a test wiring root.
func IsProvidersWiringRoot(fnIdent Identifier, paramTypes []TypeNode) bool {
	if fnIdent == "main" {
		return true
	}
	if !strings.HasPrefix(string(fnIdent), "Test") {
		return false
	}
	if len(paramTypes) == 0 {
		return true
	}
	return IsTestingTParamType(paramTypes[0])
}

// IsPublicExportIdent reports whether id is an exported (public) Forst identifier (uppercase initial).
func IsPublicExportIdent(id Identifier) bool {
	s := string(id)
	if s == "" {
		return false
	}
	return unicode.IsUpper(rune(s[0]))
}

// ParamTypesFromFunction collects parameter types from simple params (skips destructured).
func ParamTypesFromFunction(fn FunctionNode) []TypeNode {
	var types []TypeNode
	for _, p := range fn.Params {
		if sp, ok := p.(SimpleParamNode); ok {
			types = append(types, sp.Type)
		}
	}
	return types
}
