package typechecker

import (
	"fmt"
	"strings"

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
	if err := tc.validateGenericCallArgCount(sig, len(argTypes), span); err != nil {
		return sig, err
	}
	bindings := make(map[ast.TypeIdent]ast.TypeNode)
	for name, typ := range explicit {
		bindings[name] = typ
	}
	if err := tc.inferGenericBindingsFromArgs(sig, bindings, argTypes, span); err != nil {
		return sig, err
	}
	if err := tc.requireAllGenericBindings(sig, bindings, span); err != nil {
		return sig, err
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

func (tc *TypeChecker) inferGenericBindingsFromArgs(
	sig FunctionSignature,
	bindings map[ast.TypeIdent]ast.TypeNode,
	argTypes [][]ast.TypeNode,
	span ast.SourceSpan,
) error {
	fixed, elem, variadic := functionHasVariadicTail(sig.Parameters)
	nFixed := len(sig.Parameters)
	if variadic {
		nFixed = fixed
	}
	if err := typeinfer.InferFromParams(nFixed, func(i int) error {
		if i >= len(sig.Parameters) || i >= len(argTypes) || len(argTypes[i]) != 1 {
			return fmt.Errorf("argument %d: expected a single type for generic inference", i)
		}
		return tc.unifyTypeParam(bindings, sig.Parameters[i].Type, argTypes[i][0], span)
	}); err != nil {
		return err
	}
	if !variadic || len(argTypes) <= fixed {
		return nil
	}
	for j := fixed; j < len(argTypes); j++ {
		if len(argTypes[j]) != 1 {
			return fmt.Errorf("argument %d: expected a single type for generic inference", j)
		}
		argTy := argTypes[j][0]
		if j == len(argTypes)-1 && argTy.IsSlice() && len(argTy.TypeParams) == 1 {
			argTy = argTy.TypeParams[0]
		}
		if err := tc.unifyTypeParam(bindings, elem, argTy, span); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TypeChecker) requireAllGenericBindings(sig FunctionSignature, bindings map[ast.TypeIdent]ast.TypeNode, span ast.SourceSpan) error {
	var unbound []string
	for _, tp := range sig.TypeParams {
		if _, ok := bindings[ast.TypeIdent(tp.Name)]; !ok {
			unbound = append(unbound, string(tp.Name))
		}
	}
	if len(unbound) == 0 {
		return nil
	}
	hint := fmt.Sprintf("%s[%s]", sig.Ident.ID, unbound[0])
	if len(unbound) > 1 {
		args := make([]string, len(unbound))
		copy(args, unbound)
		hint = fmt.Sprintf("%s[%s]", sig.Ident.ID, strings.Join(args, ", "))
	}
	return tc.genericDiag(span, fmt.Sprintf(
		"function %s: could not infer type argument(s) %s; try explicit type arguments such as %s(...)",
		sig.Ident.ID, strings.Join(unbound, ", "), hint,
	))
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

	if param.Ident == ast.TypeFunc && arg.Ident == ast.TypeFunc {
		return tc.unifyFuncTypes(bindings, param, arg, span)
	}

	if param.Ident != "" && param.Ident == arg.Ident {
		if param.Ident == ast.TypeArray && !arrayLengthsCompatible(param, arg) {
			return tc.genericDiag(span, fmt.Sprintf("cannot unify parameter type %s with argument type %s", param.String(), arg.String()))
		}
		if len(param.TypeParams) > 0 {
			if len(param.TypeParams) != len(arg.TypeParams) {
				return tc.genericDiag(span, fmt.Sprintf("cannot unify parameter type %s with argument type %s", param.String(), arg.String()))
			}
			for i := range param.TypeParams {
				if err := tc.unifyTypeParam(bindings, param.TypeParams[i], arg.TypeParams[i], span); err != nil {
					return err
				}
			}
			return nil
		}
	}

	if ok, err := tc.unifyShapeTypes(bindings, param, arg, span); err != nil {
		return err
	} else if ok {
		return nil
	}

	return nil
}

func (tc *TypeChecker) unifyFuncTypes(bindings map[ast.TypeIdent]ast.TypeNode, param, arg ast.TypeNode, span ast.SourceSpan) error {
	if len(param.FuncParams) != len(arg.FuncParams) {
		return tc.genericDiag(span, fmt.Sprintf("cannot unify parameter type %s with argument type %s", param.String(), arg.String()))
	}
	for i := range param.FuncParams {
		pParam, okP := param.FuncParams[i].(ast.SimpleParamNode)
		aParam, okA := arg.FuncParams[i].(ast.SimpleParamNode)
		if !okP || !okA {
			continue
		}
		if err := tc.unifyTypeParam(bindings, pParam.Type, aParam.Type, span); err != nil {
			return err
		}
	}
	if len(param.FuncReturns) != len(arg.FuncReturns) {
		return tc.genericDiag(span, fmt.Sprintf("cannot unify parameter type %s with argument type %s", param.String(), arg.String()))
	}
	for i := range param.FuncReturns {
		if err := tc.unifyTypeParam(bindings, param.FuncReturns[i], arg.FuncReturns[i], span); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TypeChecker) unifyShapeTypes(bindings map[ast.TypeIdent]ast.TypeNode, param, arg ast.TypeNode, span ast.SourceSpan) (bool, error) {
	paramFields, okParam := tc.ShapeFieldsFromParamType(param)
	argFields, okArg := tc.ShapeFieldsFromParamType(arg)
	if !okParam || !okArg {
		return false, nil
	}
	if len(paramFields) != len(argFields) {
		return false, tc.genericDiag(span, fmt.Sprintf("cannot unify parameter type %s with argument type %s", param.String(), arg.String()))
	}
	for name, pf := range paramFields {
		af, ok := argFields[name]
		if !ok {
			return false, tc.genericDiag(span, fmt.Sprintf("cannot unify parameter type %s with argument type %s", param.String(), arg.String()))
		}
		pt, okP := ShapeFieldTypeNode(pf)
		at, okA := ShapeFieldTypeNode(af)
		if !okP || !okA {
			continue
		}
		if err := tc.unifyTypeParam(bindings, pt, at, span); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (tc *TypeChecker) validateGenericCallArgCount(sig FunctionSignature, nArgs int, span ast.SourceSpan) error {
	fixed, _, variadic := functionHasVariadicTail(sig.Parameters)
	nParams := len(sig.Parameters)
	if !variadic {
		if nArgs != nParams {
			return tc.genericDiag(span, fmt.Sprintf("function %s expects %d arguments, got %d", sig.Ident.ID, nParams, nArgs))
		}
		return nil
	}
	if nArgs < fixed {
		return tc.genericDiag(span, fmt.Sprintf("function %s expects at least %d arguments, got %d", sig.Ident.ID, fixed, nArgs))
	}
	return nil
}

func (tc *TypeChecker) genericDiag(span ast.SourceSpan, msg string) error {
	if span.IsSet() {
		return diagnosticf(span, "generic-type", "%s", msg)
	}
	return fmt.Errorf("%s", msg)
}
