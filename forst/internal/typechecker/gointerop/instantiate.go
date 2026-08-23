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
		_ = i
		unifyType(bindings, params.At(i).Type(), arg)
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
		if c := tp.Constraint(); c != nil {
			if inferred := inferFromConstraint(c, bindings); inferred != nil {
				targs[i] = inferred
				continue
			}
			targs[i] = c
		} else {
			targs[i] = types.Typ[types.Invalid]
		}
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

func unifyType(bindings map[*types.TypeParam]types.Type, param, arg types.Type) {
	if param == nil || arg == nil {
		return
	}
	param = types.Unalias(param)
	arg = types.Unalias(arg)

	if tp, ok := param.(*types.TypeParam); ok {
		if existing, ok := bindings[tp]; ok {
			if !types.Identical(existing, arg) && !types.AssignableTo(arg, existing) {
				// prefer concrete arg type
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
