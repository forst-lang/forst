package typechecker

import (
	"forst/internal/ast"
)

func (tc *TypeChecker) inferFunctionLiteral(lit ast.FunctionLiteralNode, scopeNode ast.Node) ([]ast.TypeNode, error) {
	tc.pushScope(scopeNode)
	defer tc.popScope()
	restoreLabels := tc.pushLoopLabelScope()
	defer restoreLabels()

	prevCapturing := tc.capturingClosure
	prevPending := tc.pendingClosureWrites
	tc.capturingClosure = true
	tc.pendingClosureWrites = nil
	defer func() {
		tc.capturingClosure = prevCapturing
		if !prevCapturing {
			// Leave pending writes for bindClosureCaptures on assignment.
		} else {
			tc.pendingClosureWrites = append(prevPending, tc.pendingClosureWrites...)
		}
	}()

	for _, param := range lit.Params {
		switch typedParam := param.(type) {
		case ast.SimpleParamNode:
			tc.scopeStack.currentScope().RegisterSymbol(
				typedParam.Ident.ID,
				[]ast.TypeNode{typedParam.Type},
				SymbolVariable,
			)
		case ast.DestructuredParamNode:
			tc.registerDestructuredParamSymbols(typedParam.Fields, typedParam.Type, SymbolVariable)
		}
	}

	fn := ast.FunctionNode{
		ReturnTypes: lit.ReturnTypes,
		Params:      lit.Params,
		Body:        lit.Body,
	}
	for _, bodyNode := range lit.Body {
		if _, err := tc.inferNodeType(bodyNode); err != nil {
			return nil, err
		}
	}
	if err := tc.checkFunctionLabels(lit.Body); err != nil {
		return nil, err
	}

	inferredReturns, err := tc.inferFunctionReturnType(fn)
	if err != nil {
		return nil, err
	}
	if len(lit.ReturnTypes) > 0 {
		if _, err := ensureMatching(tc, fn, inferredReturns, lit.ReturnTypes, "Function literal return type mismatch"); err != nil {
			return nil, err
		}
	} else if len(inferredReturns) > 0 {
		lit.ReturnTypes = inferredReturns
	}

	fnType := ast.NewFunctionType(lit.Params, lit.ReturnTypes)
	if len(lit.ReturnTypes) == 0 && len(inferredReturns) > 0 {
		fnType = ast.NewFunctionType(lit.Params, inferredReturns)
	}
	return []ast.TypeNode{fnType}, nil
}

func (tc *TypeChecker) inferCalleeCall(
	callee ast.ExpressionNode,
	args []ast.ExpressionNode,
	argSpans []ast.SourceSpan,
	callSpan ast.SourceSpan,
) ([]ast.TypeNode, error) {
	calleeTypes, err := tc.inferExpressionType(callee)
	if err != nil {
		return nil, err
	}
	if len(calleeTypes) != 1 || !calleeTypes[0].IsFunctionType() {
		sp := callSpan
		if !sp.IsSet() && callee != nil {
			if vn, ok := callee.(ast.VariableNode); ok {
				sp = vn.Ident.Span
			}
		}
		return nil, reportBodyf(sp, "type-error", "cannot call non-function value of type %s", formatTypeList(calleeTypes))
	}
	fnType := calleeTypes[0]
	if err := tc.checkFunctionTypeCall(fnType, args, argSpans, callSpan); err != nil {
		return nil, err
	}
	if vn, ok := callee.(ast.VariableNode); ok {
		sp := callSpan
		if !sp.IsSet() {
			sp = vn.Ident.Span
		}
		tc.applyClosureCallInvalidation(vn.Ident.ID, sp)
	}
	return append([]ast.TypeNode(nil), fnType.FuncReturns...), nil
}

func (tc *TypeChecker) checkFunctionTypeCall(
	fnType ast.TypeNode,
	args []ast.ExpressionNode,
	argSpans []ast.SourceSpan,
	callSpan ast.SourceSpan,
) error {
	params := fnType.FuncParams
	if len(args) != len(params) {
		return reportBodyf(callSpan, "type-error", "function call expects %d arguments, got %d", len(params), len(args))
	}
	for i, arg := range args {
		param, ok := params[i].(ast.SimpleParamNode)
		if !ok {
			continue
		}
		argTypes, err := tc.inferExpressionTypeWithExpected(arg, &param.Type)
		if err != nil {
			return err
		}
		if len(argTypes) != 1 || !tc.IsTypeCompatible(argTypes[0], param.Type) {
			sp := callSpan
			if i < len(argSpans) && argSpans[i].IsSet() {
				sp = argSpans[i]
			}
			return reportBodyf(sp, "type-error", "argument %d type %s is not compatible with parameter type %s",
				i+1, formatTypeList(argTypes), param.Type.String())
		}
	}
	return nil
}

func (tc *TypeChecker) checkFunctionTypeCompatible(actual, expected ast.TypeNode) bool {
	if !actual.IsFunctionType() || !expected.IsFunctionType() {
		return false
	}
	if len(actual.FuncParams) != len(expected.FuncParams) {
		return false
	}
	for i := range actual.FuncParams {
		act, ok1 := actual.FuncParams[i].(ast.SimpleParamNode)
		exp, ok2 := expected.FuncParams[i].(ast.SimpleParamNode)
		if !ok1 || !ok2 {
			return false
		}
		if !tc.IsTypeCompatible(act.Type, exp.Type) || !tc.IsTypeCompatible(exp.Type, act.Type) {
			return false
		}
	}
	if len(actual.FuncReturns) != len(expected.FuncReturns) {
		return false
	}
	for i := range actual.FuncReturns {
		if !tc.IsTypeCompatible(actual.FuncReturns[i], expected.FuncReturns[i]) ||
			!tc.IsTypeCompatible(expected.FuncReturns[i], actual.FuncReturns[i]) {
			return false
		}
	}
	return true
}
