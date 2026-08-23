package typechecker

import (
	"fmt"

	"forst/internal/ast"
	"forst/internal/typechecker/typeinfer"
)

func (tc *TypeChecker) IsTypeParamType(t ast.TypeNode) bool {
	return t.IsTypeParam()
}

func (tc *TypeChecker) instantiateGenericCall(sig FunctionSignature, argTypes [][]ast.TypeNode, span ast.SourceSpan) (FunctionSignature, error) {
	return tc.instantiateGenericCallWithBindings(sig, nil, argTypes, span)
}

func (tc *TypeChecker) instantiateGenericCallWithBindings(
	sig FunctionSignature,
	explicit map[ast.TypeIdent]ast.TypeNode,
	argTypes [][]ast.TypeNode,
	span ast.SourceSpan,
) (FunctionSignature, error) {
	if len(sig.TypeParams) == 0 {
		return sig, nil
	}
	bindings := make(map[ast.TypeIdent]ast.TypeNode)
	for name, typ := range explicit {
		bindings[name] = typ
	}
	paramCount := len(sig.Parameters)
	if len(argTypes) < paramCount {
		paramCount = len(argTypes)
	}
	err := typeinfer.InferFromParams(paramCount, func(i int) error {
		if i >= len(sig.Parameters) {
			return nil
		}
		if i >= len(argTypes) || len(argTypes[i]) != 1 {
			return fmt.Errorf("argument %d: expected a single type for generic inference", i)
		}
		return tc.unifyTypeParam(bindings, sig.Parameters[i].Type, argTypes[i][0], span)
	})
	if err != nil {
		return sig, err
	}
	if err := typeinfer.RequireAllBound(len(sig.TypeParams), func(i int) bool {
		tp := sig.TypeParams[i]
		_, ok := bindings[ast.TypeIdent(tp.Name)]
		return ok
	}, func(i int) string {
		return string(sig.TypeParams[i].Name)
	}); err != nil {
		return sig, tc.genericDiag(span, err.Error())
	}
	if err := tc.checkTypeParamConstraints(sig, bindings, span); err != nil {
		return sig, err
	}
	return tc.substituteFunctionSignature(sig, bindings), nil
}

func (tc *TypeChecker) instantiateGenericCallExplicit(
	sig FunctionSignature,
	typeArgs []ast.TypeNode,
	argTypes [][]ast.TypeNode,
	span ast.SourceSpan,
) (FunctionSignature, error) {
	if len(typeArgs) != len(sig.TypeParams) {
		return sig, tc.genericDiag(span, fmt.Sprintf("%s: expected %d type arguments, got %d", sig.Ident.ID, len(sig.TypeParams), len(typeArgs)))
	}
	explicit := make(map[ast.TypeIdent]ast.TypeNode, len(typeArgs))
	for i, ta := range typeArgs {
		explicit[ast.TypeIdent(sig.TypeParams[i].Name)] = ta
	}
	return tc.instantiateGenericCallWithBindings(sig, explicit, argTypes, span)
}

func (tc *TypeChecker) substituteFunctionSignature(sig FunctionSignature, bindings map[ast.TypeIdent]ast.TypeNode) FunctionSignature {
	inst := sig
	inst.Parameters = make([]ParameterSignature, len(sig.Parameters))
	for i, p := range sig.Parameters {
		inst.Parameters[i] = ParameterSignature{
			Ident:    p.Ident,
			Type:     tc.substituteTypeBindings(p.Type, bindings),
			Variadic: p.Variadic,
		}
	}
	inst.ReturnTypes = make([]ast.TypeNode, len(sig.ReturnTypes))
	for i, rt := range sig.ReturnTypes {
		inst.ReturnTypes[i] = tc.substituteTypeBindings(rt, bindings)
	}
	inst.TypeParams = nil
	inst.TypeParamNames = nil
	return inst
}

func (tc *TypeChecker) unifyTypeParam(bindings map[ast.TypeIdent]ast.TypeNode, param, arg ast.TypeNode, span ast.SourceSpan) error {
	if tc.IsTypeParamType(param) {
		name := param.Ident
		if existing, ok := bindings[name]; ok {
			if tc.IsTypeParamType(existing) {
				bindings[name] = arg
				return nil
			}
			if !tc.IsTypeCompatible(arg, existing) && !tc.IsTypeCompatible(existing, arg) {
				return tc.genericDiag(span, fmt.Sprintf("type argument %s inferred as %s and %s", name, existing.String(), arg.String()))
			}
			return nil
		}
		bindings[name] = arg
		return nil
	}
	if param.Ident != "" && param.Ident == arg.Ident && len(param.TypeParams) > 0 && len(param.TypeParams) == len(arg.TypeParams) {
		for i := range param.TypeParams {
			if err := tc.unifyTypeParam(bindings, param.TypeParams[i], arg.TypeParams[i], span); err != nil {
				return err
			}
		}
		return nil
	}
	if param.Ident == ast.TypeArray && arg.Ident == ast.TypeArray && len(param.TypeParams) == 1 && len(arg.TypeParams) == 1 {
		return tc.unifyTypeParam(bindings, param.TypeParams[0], arg.TypeParams[0], span)
	}
	if param.Ident == ast.TypePointer && arg.Ident == ast.TypePointer && len(param.TypeParams) == 1 && len(arg.TypeParams) == 1 {
		return tc.unifyTypeParam(bindings, param.TypeParams[0], arg.TypeParams[0], span)
	}
	if param.Ident == ast.TypeMap && arg.Ident == ast.TypeMap && len(param.TypeParams) == 2 && len(arg.TypeParams) == 2 {
		if err := tc.unifyTypeParam(bindings, param.TypeParams[0], arg.TypeParams[0], span); err != nil {
			return err
		}
		return tc.unifyTypeParam(bindings, param.TypeParams[1], arg.TypeParams[1], span)
	}
	return nil
}

func (tc *TypeChecker) genericDiag(span ast.SourceSpan, msg string) error {
	if span.IsSet() {
		return diagnosticf(span, "generic-type", "%s", msg)
	}
	return fmt.Errorf("%s", msg)
}
