package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

func functionHasVariadicTail(params []ParameterSignature) (fixed int, elem ast.TypeNode, ok bool) {
	if len(params) == 0 {
		return 0, ast.TypeNode{}, false
	}
	last := params[len(params)-1]
	if !last.Variadic {
		return 0, ast.TypeNode{}, false
	}
	return len(params) - 1, last.Type, true
}

func expectedTypeForCallParam(params []ParameterSignature, argIndex int) *ast.TypeNode {
	if len(params) == 0 {
		return nil
	}
	fixed, elem, variadic := functionHasVariadicTail(params)
	var pt ast.TypeNode
	switch {
	case variadic && argIndex >= fixed:
		pt = elem
	case argIndex < len(params):
		pt = params[argIndex].Type
	default:
		return nil
	}
	if pt.IsTypeParam() {
		return nil
	}
	return &pt
}

func (tc *TypeChecker) checkUserFunctionCall(fn ast.Identifier, sig FunctionSignature, e ast.FunctionCallNode, argTypes [][]ast.TypeNode) error {
	fixed, elem, variadic := functionHasVariadicTail(sig.Parameters)
	nArgs := len(argTypes)

	if !variadic {
		if nArgs != len(sig.Parameters) {
			sp := e.CallSpan
			if nArgs > len(sig.Parameters) {
				sp = spanForCallArg(e.ArgSpans, len(sig.Parameters), e.Arguments, e.CallSpan)
			}
			if !sp.IsSet() {
				sp = e.Function.Span
			}
			return reportBodyf(sp, "call-arity", "function %s expects %d arguments, got %d",
				fn, len(sig.Parameters), nArgs)
		}
		for i, param := range sig.Parameters {
			if err := tc.checkUserCallArg(fn, i, param.Type, argTypes[i], e); err != nil {
				return err
			}
		}
		if err := tc.checkCallFactRequirements(fn, e); err != nil {
			return err
		}
		tc.applyCallSummaryInvalidation(fn, e)
		for _, arg := range e.Arguments {
			sp := e.CallSpan
			if !sp.IsSet() {
				sp = e.Function.Span
			}
			tc.applyClosureEscapeInvalidation(arg, sp)
		}
		return nil
	}

	if nArgs < fixed {
		sp := e.CallSpan
		if !sp.IsSet() {
			sp = e.Function.Span
		}
		return reportBodyf(sp, "call-arity", "function %s expects at least %d arguments, got %d",
			fn, fixed, nArgs)
	}

	for i := 0; i < fixed; i++ {
		if err := tc.checkUserCallArg(fn, i, sig.Parameters[i].Type, argTypes[i], e); err != nil {
			return err
		}
	}

	if nArgs == fixed {
		return nil
	}

	if nArgs > fixed {
		if spread, isSpread := e.Arguments[nArgs-1].(ast.SpreadExpressionNode); isSpread {
			if nArgs != fixed+1 {
				sp := spanForCallArg(e.ArgSpans, fixed+1, e.Arguments, e.CallSpan)
				return reportBodyf(sp, "call-arity", "function %s: variadic spread must be the only trailing argument", fn)
			}
			spreadTypes, err := tc.inferExpressionType(spread.Expr)
			if err != nil {
				return err
			}
			wantSlice := ast.NewArrayType(elem)
			if len(spreadTypes) != 1 || !tc.IsTypeCompatible(spreadTypes[0], wantSlice) {
				sp := spanForCallArg(e.ArgSpans, fixed, e.Arguments, e.CallSpan)
				return reportBodyf(sp, "call-type", "function %s: cannot spread %s into ...%s",
					fn, formatTypeForDiag(spreadTypes), elem.Ident)
			}
			return nil
		}
		for j := fixed; j < nArgs; j++ {
			if err := tc.checkUserCallArg(fn, j, elem, argTypes[j], e); err != nil {
				return err
			}
		}
	}
	return nil
}

func formatTypeForDiag(types []ast.TypeNode) string {
	if len(types) == 0 {
		return "?"
	}
	if len(types) == 1 {
		return formatTypeNodeForDiag(types[0])
	}
	return fmt.Sprintf("%d types", len(types))
}

func (tc *TypeChecker) checkUserCallArg(fn ast.Identifier, argIdx int, want ast.TypeNode, got []ast.TypeNode, e ast.FunctionCallNode) error {
	sp := spanForCallArg(e.ArgSpans, argIdx, e.Arguments, e.CallSpan)
	if len(got) != 1 {
		return reportBodyf(sp, "call-type", "argument %d to %s must have a single type, got %d",
			argIdx+1, fn, len(got))
	}
	if !tc.IsTypeCompatible(got[0], want) {
		return reportBodyf(sp, "call-type", "argument %d to %s: expected type %s, got %s",
			argIdx+1, fn, formatTypeIdentForDiag(want.Ident), formatTypeIdentForDiag(got[0].Ident))
	}
	return nil
}
