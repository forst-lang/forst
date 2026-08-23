package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

func (tc *TypeChecker) IsTypeParamType(t ast.TypeNode) bool {
	if t.IsTypeParam() {
		return true
	}
	if t.Ident == "" {
		return false
	}
	def, ok := tc.Defs[t.Ident]
	if !ok {
		return false
	}
	tn, ok := def.(ast.TypeNode)
	return ok && tn.IsTypeParam()
}

func (tc *TypeChecker) instantiateGenericCall(sig FunctionSignature, argTypes [][]ast.TypeNode) (FunctionSignature, error) {
	if len(sig.TypeParams) == 0 {
		return sig, nil
	}
	bindings := make(map[ast.TypeIdent]ast.TypeNode)
	for i, param := range sig.Parameters {
		if i >= len(argTypes) || len(argTypes[i]) != 1 {
			continue
		}
		tc.unifyTypeParam(bindings, param.Type, argTypes[i][0])
	}
	for _, tp := range sig.TypeParams {
		if _, ok := bindings[ast.TypeIdent(tp.Name)]; !ok {
			return sig, fmt.Errorf("could not infer type argument %s", tp.Name)
		}
	}
	inst := sig
	inst.Parameters = make([]ParameterSignature, len(sig.Parameters))
	for i, p := range sig.Parameters {
		inst.Parameters[i] = ParameterSignature{
			Ident:    p.Ident,
			Type:     tc.substituteTypeParams(p.Type, bindings),
			Variadic: p.Variadic,
		}
	}
	inst.ReturnTypes = make([]ast.TypeNode, len(sig.ReturnTypes))
	for i, rt := range sig.ReturnTypes {
		inst.ReturnTypes[i] = tc.substituteTypeParams(rt, bindings)
	}
	inst.TypeParams = nil
	return inst, nil
}

func (tc *TypeChecker) unifyTypeParam(bindings map[ast.TypeIdent]ast.TypeNode, param, arg ast.TypeNode) {
	if tc.IsTypeParamType(param) {
		name := param.Ident
		if existing, ok := bindings[name]; ok {
			if !tc.IsTypeCompatible(arg, existing) && !tc.IsTypeCompatible(existing, arg) {
				if tc.IsTypeParamType(existing) {
					bindings[name] = arg
				}
			}
			return
		}
		bindings[name] = arg
		return
	}
	if param.Ident != "" && param.Ident == arg.Ident && len(param.TypeParams) > 0 && len(param.TypeParams) == len(arg.TypeParams) {
		for i := range param.TypeParams {
			tc.unifyTypeParam(bindings, param.TypeParams[i], arg.TypeParams[i])
		}
	}
}

func (tc *TypeChecker) substituteTypeParams(t ast.TypeNode, bindings map[ast.TypeIdent]ast.TypeNode) ast.TypeNode {
	if tc.IsTypeParamType(t) {
		if bound, ok := bindings[t.Ident]; ok {
			return bound
		}
		return t
	}
	if len(t.TypeParams) == 0 {
		return t
	}
	out := t
	out.TypeParams = make([]ast.TypeNode, len(t.TypeParams))
	for i, p := range t.TypeParams {
		out.TypeParams[i] = tc.substituteTypeParams(p, bindings)
	}
	return out
}
