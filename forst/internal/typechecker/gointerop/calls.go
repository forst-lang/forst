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
		pkgName := c.QualDisplay
		if pkgName == "" || pkgName == c.FuncName {
			pkgName = c.Pkg.Name()
		}
		return nil, &MemberMissingError{
			Span:    sp,
			Pkg:     pkgName,
			Member:  c.FuncName,
			Exports: exportedNames(c.Pkg.Scope()),
		}
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		sp := c.Call.Function.Span
		if !sp.IsSet() {
			sp = c.Call.CallSpan
		}
		return nil, diag(sp, "go-call",
			fmt.Sprintf("`%s` is not a function", qual),
			fmt.Sprintf("`%s` names a non-function symbol in the Go package.", qual),
			"call a function or import the correct exported name")
	}
	if c.RequireExported && !fn.Exported() {
		sp := c.Call.Function.Span
		if !sp.IsSet() {
			sp = c.Call.CallSpan
		}
		return nil, diag(sp, "go-call",
			fmt.Sprintf("`%s` is not exported", qual),
			fmt.Sprintf("Symbol `%s` exists but is unexported in its Go package.", qual),
			"use an exported name or call from the same Go package")
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, diag(c.Call.CallSpan, "go-call",
			fmt.Sprintf("`%s` has an invalid signature", qual),
			"Could not read the Go function signature.",
			"check the Go definition")
	}
	if sig.TypeParams() != nil && sig.TypeParams().Len() > 0 {
		argGoTypes := GoTypesFromForstArgs(host, c.ArgTypes)
		instSig, err := InstantiateFuncSignature(fn, argGoTypes)
		if err != nil {
			sp := c.Call.Function.Span
			if !sp.IsSet() {
				sp = c.Call.CallSpan
			}
			return nil, diag(sp, "go-call",
				fmt.Sprintf("`%s` could not be instantiated", qual),
				fmt.Sprintf("Generic instantiation failed: %v", err),
				"pass arguments whose types satisfy the Go generic constraints")
		}
		sig = instSig
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
			return nil, diag(c.Call.CallSpan, "go-call",
				fmt.Sprintf("`%s` expects at least %d arguments", c.Qual, fixed),
				fmt.Sprintf("Got %d argument(s).", nArgs),
				"add the missing arguments before the variadic parameter")
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
			return nil, diag(c.Call.CallSpan, "go-call",
				fmt.Sprintf("`%s` has an invalid variadic parameter", c.Qual),
				"The last parameter is not a valid Go slice type.",
				"check the Go function signature")
		}
		elem := sliceT.Elem()
		if nArgs > fixed {
			if spread, isSpread := c.Call.Arguments[nArgs-1].(ast.SpreadExpressionNode); isSpread {
				if nArgs != fixed+1 {
					sp := spanForCallArg(c.Call.ArgSpans, fixed+1, c.Call.Arguments, c.Call.CallSpan)
					return nil, diag(sp, "go-call",
						"variadic spread must be last",
						fmt.Sprintf("`%s` accepts `...%s` as its only trailing argument.", c.Qual, elem.String()),
						"pass one spread argument after the fixed parameters")
				}
				spreadTypes, err := host.InferExpressionType(spread.Expr)
				if err != nil {
					return nil, err
				}
				if len(spreadTypes) != 1 {
					return nil, diag(spanForCallArg(c.Call.ArgSpans, fixed, c.Call.Arguments, c.Call.CallSpan), "go-call",
						"spread argument has multiple types",
						fmt.Sprintf("`%s` variadic spread expects one concrete type.", c.Qual),
						"ensure the spread expression has a single inferred type")
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
			return nil, diag(sp, "go-call",
				fmt.Sprintf("`%s` expects %d arguments", c.Qual, nParams),
				fmt.Sprintf("Got %d argument(s).", nArgs),
				"add or remove arguments to match the Go signature")
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
		gt, ok := MapGoType(host, res.At(i).Type())
		if !ok {
			sp := c.Call.Function.Span
			if !sp.IsSet() {
				sp = c.Call.CallSpan
			}
			goRet := res.At(i).Type().String()
			return nil, diag(sp, "go-call",
				"unsupported Go return type",
				fmt.Sprintf("`%s` returns `%s`, which Forst cannot map.", c.Qual, goRet),
				"wrap the result in a supported type or avoid calling this API from Forst")
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
		return diag(sp, "go-call",
			fmt.Sprintf("argument %d has multiple types", p.Index+1),
			fmt.Sprintf("`%s` parameter %d expects one type, got %d.", p.Qual, p.Index+1, len(p.ArgType)),
			"resolve the argument to a single concrete type")
	}
	if p.ArgIdx >= 0 && p.ArgIdx < len(p.Call.Arguments) {
		if argGo := host.GoTypeForExpression(p.Call.Arguments[p.ArgIdx]); argGo != nil {
			if types.AssignableTo(argGo, p.GoParam) {
				return nil
			}
		}
	}
	if !ForstAssignableToGoType(host, p.ArgType[0], p.GoParam) {
		goParam := strings.TrimSpace(p.GoParam.String())
		return diag(sp, "go-call",
			fmt.Sprintf("argument %d type mismatch", p.Index+1),
			fmt.Sprintf("Forst type `%s` is not assignable to Go parameter `%s`.", p.ArgType[0].Ident, goParam),
			"convert the argument or change the Forst type")
	}
	return nil
}

// CheckSpreadAssignability validates a variadic spread argument.
func CheckSpreadAssignability(host Host, diag Diagnose, s SpreadAssignability) error {
	if s.SpreadType.Ident != ast.TypeArray || len(s.SpreadType.TypeParams) != 1 {
		sp := spanForCallArg(s.Call.ArgSpans, s.ArgIdx, s.Call.Arguments, s.Call.CallSpan)
		return diag(sp, "go-call",
			"spread requires a slice",
			fmt.Sprintf("`%s` variadic parameter expects `...%s`, but spread has type `%s`.", s.Qual, s.Elem.String(), s.SpreadType.Ident),
			"pass a slice or use individual arguments")
	}
	wantSlice := types.NewSlice(s.Elem)
	if !ForstAssignableToGoType(host, s.SpreadType, wantSlice) {
		sp := spanForCallArg(s.Call.ArgSpans, s.ArgIdx, s.Call.Arguments, s.Call.CallSpan)
		return diag(sp, "go-call",
			"cannot spread into variadic parameter",
			fmt.Sprintf("Slice `%s` is not assignable to `...%s`.", s.SpreadType.Ident, s.Elem.String()),
			"use a slice of the correct element type")
	}
	return nil
}

func methodDiagSpan(m MethodCall) ast.SourceSpan {
	if m.Method.Span.IsSet() {
		return m.Method.Span
	}
	if m.Call.CallSpan.IsSet() {
		return m.Call.CallSpan
	}
	return m.Call.Function.Span
}

// CheckMethodCall type-checks a Go method call when the receiver has a tracked go/types type.
func CheckMethodCall(host Host, diag Diagnose, m MethodCall) ([]ast.TypeNode, error) {
	obj, _, _ := types.LookupFieldOrMethod(m.Recv, true, nil, m.MethodName)
	if obj == nil {
		sp := methodDiagSpan(m)
		return nil, diag(sp, "go-method",
			fmt.Sprintf("`%s` has no field or method `%s`", m.Recv.String(), m.MethodName),
			fmt.Sprintf("Type `%s` does not define `%s`.", m.Recv.String(), m.MethodName),
			"check the spelling or call a method that exists on the receiver type")
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		sp := methodDiagSpan(m)
		return nil, diag(sp, "go-method",
			fmt.Sprintf("`%s` is not a method", m.MethodName),
			fmt.Sprintf("`%s` names a field or value on `%s`, not a method.", m.MethodName, m.Recv.String()),
			"call a method on the receiver or access the field without parentheses")
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, diag(m.Call.CallSpan, "go-method",
			"invalid method signature",
			"Could not read the Go method signature.",
			"check the Go definition")
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
