package gointerop

import (
	"fmt"

	"go/types"

	"forst/internal/ast"
	"forst/internal/typechecker/typeinfer"
)

// InstantiateFuncSignature resolves type arguments for a generic Go function and returns
// the instantiated signature, or an error when inference fails.
func InstantiateFuncSignature(fn *types.Func, argGoTypes []types.Type) (*types.Signature, error) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("invalid signature")
	}
	tparams := sig.TypeParams()
	if tparams == nil || tparams.Len() == 0 {
		return sig, nil
	}

	targs, err := inferTypeArgs(sig, argGoTypes)
	if err != nil {
		return nil, err
	}
	if len(targs) != tparams.Len() {
		return nil, fmt.Errorf("could not infer %d type arguments", tparams.Len())
	}

	inst, err := types.Instantiate(nil, sig, targs, true)
	if err != nil {
		return nil, err
	}
	instSig, ok := inst.(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("instantiated type is not a signature")
	}
	return instSig, nil
}

func inferTypeArgs(sig *types.Signature, argGoTypes []types.Type) ([]types.Type, error) {
	tparams := sig.TypeParams()
	if tparams == nil || tparams.Len() == 0 {
		return nil, nil
	}
	bindings := make(map[*types.TypeParam]types.Type)
	params := sig.Params()
	n := params.Len()
	if len(argGoTypes) < n && !sig.Variadic() {
		return nil, fmt.Errorf("not enough arguments for type inference")
	}
	for i := 0; i < n; i++ {
		var arg types.Type
		if i < len(argGoTypes) {
			arg = argGoTypes[i]
		}
		if sig.Variadic() && i == n-1 {
			if sl, ok := arg.Underlying().(*types.Slice); ok {
				arg = sl.Elem()
			}
		}
		unifyType(bindings, params.At(i).Type(), arg)
	}
	inferDependentTypeParams(tparams, bindings)
	for i := 0; i < tparams.Len(); i++ {
		tp := tparams.At(i)
		if _, ok := bindings[tp]; ok {
			continue
		}
		if c := tp.Constraint(); c != nil {
			if inferred := inferFromConstraint(c, bindings); inferred != nil {
				bindings[tp] = inferred
			}
		}
	}
	if err := typeinfer.RequireAllBound(tparams.Len(), func(i int) bool {
		tp := tparams.At(i)
		_, ok := bindings[tp]
		return ok
	}, func(i int) string {
		return tparams.At(i).Obj().Name()
	}); err != nil {
		return nil, err
	}
	targs := make([]types.Type, tparams.Len())
	for i := 0; i < tparams.Len(); i++ {
		tp := tparams.At(i)
		if bound, ok := bindings[tp]; ok {
			targs[i] = bound
			continue
		}
		targs[i] = types.Typ[types.Invalid]
	}
	for i, ta := range targs {
		if ta == types.Typ[types.Invalid] {
			return nil, fmt.Errorf("could not infer type argument %d (%s)", i, tparams.At(i).Obj().Name())
		}
	}
	return targs, nil
}

// inferFromConstraint derives a type argument from a constraint and existing bindings (e.g. E from ~[]E when S is bound).
func inferFromConstraint(c types.Type, bindings map[*types.TypeParam]types.Type) types.Type {
	c = types.Unalias(c)
	if u, ok := c.(*types.Union); ok && u.Len() == 1 {
		c = u.Term(0).Type()
		c = types.Unalias(c)
	}
	if iface, ok := c.Underlying().(*types.Interface); ok {
		for i := 0; i < iface.NumEmbeddeds(); i++ {
			emb := iface.EmbeddedType(i)
			if sl, ok := emb.Underlying().(*types.Slice); ok {
				elem := sl.Elem()
				if tp, ok := elem.(*types.TypeParam); ok {
					if bound, ok := bindings[tp]; ok {
						return bound
					}
				}
			}
		}
	}
	return nil
}

func inferDependentTypeParams(tparams *types.TypeParamList, bindings map[*types.TypeParam]types.Type) {
	changed := true
	for changed {
		changed = false
		for i := 0; i < tparams.Len(); i++ {
			target := tparams.At(i)
			if _, ok := bindings[target]; ok {
				continue
			}
			for j := 0; j < tparams.Len(); j++ {
				source := tparams.At(j)
				sourceBound, ok := bindings[source]
				if !ok || source.Constraint() == nil {
					continue
				}
				if inferred := inferFromBoundConstraint(target, source.Constraint(), sourceBound); inferred != nil {
					bindings[target] = inferred
					changed = true
					break
				}
			}
		}
	}
}

func inferFromBoundConstraint(target *types.TypeParam, constraint types.Type, bound types.Type) types.Type {
	for _, term := range constraintTerms(constraint) {
		sl, ok := term.Underlying().(*types.Slice)
		if !ok {
			continue
		}
		elemTP, ok := sl.Elem().(*types.TypeParam)
		if !ok || elemTP.Obj() != target.Obj() {
			continue
		}
		if boundSl, ok := bound.Underlying().(*types.Slice); ok {
			return boundSl.Elem()
		}
	}
	return nil
}

func constraintTerms(c types.Type) []types.Type {
	c = types.Unalias(c)
	if u, ok := c.(*types.Union); ok {
		out := make([]types.Type, 0, u.Len())
		for i := 0; i < u.Len(); i++ {
			out = append(out, constraintTerms(u.Term(i).Type())...)
		}
		return out
	}
	if iface, ok := c.Underlying().(*types.Interface); ok {
		out := make([]types.Type, 0, iface.NumEmbeddeds())
		for i := 0; i < iface.NumEmbeddeds(); i++ {
			out = append(out, constraintTerms(iface.EmbeddedType(i))...)
		}
		return out
	}
	return []types.Type{c}
}

func unifyType(bindings map[*types.TypeParam]types.Type, param, arg types.Type) {
	if param == nil || arg == nil {
		return
	}
	param = types.Unalias(param)
	arg = types.Unalias(arg)

	if tp, ok := param.(*types.TypeParam); ok {
		if existing, ok := bindings[tp]; ok {
			if !types.Identical(existing, arg) && !types.AssignableTo(arg, existing) {
				if _, isParam := existing.(*types.TypeParam); isParam {
					bindings[tp] = arg
				}
			}
			return
		}
		bindings[tp] = arg
		return
	}

	switch p := param.Underlying().(type) {
	case *types.Slice:
		if sl, ok := arg.Underlying().(*types.Slice); ok {
			unifyType(bindings, p.Elem(), sl.Elem())
		}
	case *types.Pointer:
		if ptr, ok := arg.Underlying().(*types.Pointer); ok {
			unifyType(bindings, p.Elem(), ptr.Elem())
		}
	case *types.Map:
		if mp, ok := arg.Underlying().(*types.Map); ok {
			unifyType(bindings, p.Key(), mp.Key())
			unifyType(bindings, p.Elem(), mp.Elem())
		}
	case *types.Array:
		if arr, ok := arg.Underlying().(*types.Array); ok {
			unifyType(bindings, p.Elem(), arr.Elem())
		}
	case *types.Chan:
		if ch, ok := arg.Underlying().(*types.Chan); ok {
			unifyType(bindings, p.Elem(), ch.Elem())
		}
	case *types.Basic:
		// concrete param: arg must match; nothing to bind
	case *types.Interface:
		// accept arg as-is for empty interface params
	default:
		if types.Identical(param, arg) {
			return
		}
	}
}

// GoTypesFromForstArgs converts Forst argument types to go/types using the host.
func GoTypesFromForstArgs(host AssignabilityHost, argTypes [][]ast.TypeNode) []types.Type {
	out := make([]types.Type, 0, len(argTypes))
	for _, at := range argTypes {
		if len(at) == 0 {
			out = append(out, types.Typ[types.Invalid])
			continue
		}
		if gt := host.GoTypeForForstType(at[0]); gt != nil {
			out = append(out, gt)
		} else {
			out = append(out, types.Typ[types.Invalid])
		}
	}
	return out
}
