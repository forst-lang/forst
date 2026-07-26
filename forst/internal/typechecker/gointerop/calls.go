package gointerop

import (
	"fmt"
	"strings"

	"go/types"

	"forst/internal/ast"
)

// CheckFuncCall type-checks a Go package-level function call.
func CheckFuncCall(host Host, diag Diagnose, c FuncCall) ([]ast.TypeNode, error) {
	qual := c.FuncName
	if c.QualDisplay != c.FuncName {
		qual = c.QualDisplay + "." + c.FuncName
	}
	obj := c.Pkg.Scope().Lookup(c.FuncName)
	if obj == nil {
		sp := c.Call.Function.Span
		if !sp.IsSet() {
			sp = c.Call.CallSpan
		}
		return nil, diag(sp, "go-call", "%s not found in Go package", qual)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		sp := c.Call.Function.Span
		if !sp.IsSet() {
			sp = c.Call.CallSpan
		}
		return nil, diag(sp, "go-call", "%s is not a function", qual)
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, diag(c.Call.CallSpan, "go-call", "%s: invalid signature", qual)
	}
	return CheckSignature(host, diag, SignatureCheck{
		Sig:             sig,
		Qual:            qual,
		Call:            c.Call,
		ArgTypes:        c.ArgTypes,
		WantSingleValue: c.WantSingleValue,
	})
}

// CheckSignature validates arguments and maps results for a Go signature.
func CheckSignature(host Host, diag Diagnose, c SignatureCheck) ([]ast.TypeNode, error) {
	params := c.Sig.Params()
	nParams := params.Len()
	nArgs := len(c.ArgTypes)

	if c.Sig.Variadic() {
		fixed := nParams - 1
		if nArgs < fixed {
			return nil, diag(c.Call.CallSpan, "go-call", "%s: expects at least %d arguments, got %d", c.Qual, fixed, nArgs)
		}
		for i := range fixed {
			if err := CheckParamAssignability(host, diag, ParamAssignability{
				Qual:    c.Qual,
				Index:   i,
				GoParam: params.At(i).Type(),
				ArgType: c.ArgTypes[i],
				Call:    c.Call,
				ArgIdx:  i,
			}); err != nil {
				return nil, err
			}
		}
		sliceT, ok := params.At(nParams - 1).Type().Underlying().(*types.Slice)
		if !ok {
			return nil, diag(c.Call.CallSpan, "go-call", "%s: invalid variadic parameter", c.Qual)
		}
		elem := sliceT.Elem()
		if nArgs > fixed {
			if spread, isSpread := c.Call.Arguments[nArgs-1].(ast.SpreadExpressionNode); isSpread {
				if nArgs != fixed+1 {
					sp := spanForCallArg(c.Call.ArgSpans, fixed+1, c.Call.Arguments, c.Call.CallSpan)
					return nil, diag(sp, "go-call", "%s: variadic spread must be the only trailing argument", c.Qual)
				}
				spreadTypes, err := host.InferExpressionType(spread.Expr)
				if err != nil {
					return nil, err
				}
				if len(spreadTypes) != 1 {
					return nil, diag(spanForCallArg(c.Call.ArgSpans, fixed, c.Call.Arguments, c.Call.CallSpan), "go-call", "%s: spread argument must have a single type", c.Qual)
				}
				if err := CheckSpreadAssignability(host, diag, SpreadAssignability{
					Qual:       c.Qual,
					Elem:       elem,
					SpreadType: spreadTypes[0],
					Call:       c.Call,
					ArgIdx:     fixed,
				}); err != nil {
					return nil, err
				}
			} else {
				for j := fixed; j < nArgs; j++ {
					if err := CheckParamAssignability(host, diag, ParamAssignability{
						Qual:    c.Qual,
						Index:   j,
						GoParam: elem,
						ArgType: c.ArgTypes[j],
						Call:    c.Call,
						ArgIdx:  j,
					}); err != nil {
						return nil, err
					}
				}
			}
		}
	} else {
		if nArgs != nParams {
			sp := c.Call.CallSpan
			if nArgs > nParams {
				sp = spanForCallArg(c.Call.ArgSpans, nParams, c.Call.Arguments, c.Call.CallSpan)
			}
			if !sp.IsSet() {
				sp = c.Call.Function.Span
			}
			return nil, diag(sp, "go-call", "%s: expects %d arguments, got %d", c.Qual, nParams, nArgs)
		}
		for i := range nParams {
			if err := CheckParamAssignability(host, diag, ParamAssignability{
				Qual:    c.Qual,
				Index:   i,
				GoParam: params.At(i).Type(),
				ArgType: c.ArgTypes[i],
				Call:    c.Call,
				ArgIdx:  i,
			}); err != nil {
				return nil, err
			}
		}
	}

	res := c.Sig.Results()
	if res.Len() == 0 {
		return []ast.TypeNode{{Ident: ast.TypeVoid}}, nil
	}
	out := make([]ast.TypeNode, res.Len())
	for i := 0; i < res.Len(); i++ {
		gt, ok := TypeToForstType(res.At(i).Type())
		if !ok {
			sp := c.Call.Function.Span
			if !sp.IsSet() {
				sp = c.Call.CallSpan
			}
			return nil, diag(sp, "go-call", "%s: unsupported Go return type %s", c.Qual, res.At(i).Type().String())
		}
		out[i] = gt
	}
	if c.WantSingleValue && res.Len() >= 2 {
		return []ast.TypeNode{ast.NewTupleType(out...)}, nil
	}
	return out, nil
}

// CheckParamAssignability validates one Forst argument against one Go parameter.
func CheckParamAssignability(host Host, diag Diagnose, p ParamAssignability) error {
	sp := spanForCallArg(p.Call.ArgSpans, p.ArgIdx, p.Call.Arguments, p.Call.CallSpan)
	if len(p.ArgType) != 1 {
		return diag(sp, "go-call", "%s argument %d must have a single type, got %d", p.Qual, p.Index+1, len(p.ArgType))
	}
	if !ForstAssignableToGoType(host, p.ArgType[0], p.GoParam) {
		return diag(sp, "go-call", "%s argument %d: Forst type %s not assignable to Go parameter %s",
			p.Qual, p.Index+1, p.ArgType[0].Ident, strings.TrimSpace(p.GoParam.String()))
	}
	return nil
}

// CheckSpreadAssignability validates a variadic spread argument.
func CheckSpreadAssignability(host Host, diag Diagnose, s SpreadAssignability) error {
	if s.SpreadType.Ident != ast.TypeArray || len(s.SpreadType.TypeParams) != 1 {
		sp := spanForCallArg(s.Call.ArgSpans, s.ArgIdx, s.Call.Arguments, s.Call.CallSpan)
		return diag(sp, "go-call", "%s: spread argument must be a slice, got %s", s.Qual, s.SpreadType.Ident)
	}
	wantSlice := types.NewSlice(s.Elem)
	if !ForstAssignableToGoType(host, s.SpreadType, wantSlice) {
		sp := spanForCallArg(s.Call.ArgSpans, s.ArgIdx, s.Call.Arguments, s.Call.CallSpan)
		return diag(sp, "go-call", "%s: cannot spread %s into ...%s", s.Qual, s.SpreadType.Ident, s.Elem.String())
	}
	return nil
}

// CheckMethodCall type-checks a Go method call when the receiver has a tracked go/types type.
func CheckMethodCall(host Host, diag Diagnose, m MethodCall) ([]ast.TypeNode, error) {
	obj, _, _ := types.LookupFieldOrMethod(m.Recv, true, nil, m.MethodName)
	if obj == nil {
		sp := m.Call.CallSpan
		if !sp.IsSet() {
			sp = m.Call.Function.Span
		}
		return nil, diag(sp, "go-method", "%s has no field or method %s", m.Recv.String(), m.MethodName)
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		sp := m.Call.CallSpan
		if !sp.IsSet() {
			sp = m.Call.Function.Span
		}
		return nil, diag(sp, "go-method", "%s.%s is not a method", m.Recv.String(), m.MethodName)
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, diag(m.Call.CallSpan, "go-method", "invalid method signature")
	}
	qual := fmt.Sprintf("(%s).%s", m.Recv.String(), m.MethodName)
	return CheckSignature(host, diag, SignatureCheck{
		Sig:             sig,
		Qual:            qual,
		Call:            m.Call,
		ArgTypes:        m.ArgTypes,
		WantSingleValue: m.WantSingleValue,
	})
}
