package typechecker

import "forst/internal/ast"

func (tc *TypeChecker) inferMakeNewTypeArg(args []ast.ExpressionNode, i int, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) (ast.TypeNode, error) {
	if i < 0 || i >= len(args) {
		return ast.TypeNode{}, reportBodyf(callSpan, "builtin-call", "internal: missing type argument")
	}
	te, ok := args[i].(ast.TypeExpressionNode)
	if !ok {
		sp := spanForCallArg(argSpans, i, args, callSpan)
		return ast.TypeNode{}, reportBodyf(sp, "builtin-call",
			"first argument must be a Forst type (e.g. Array(Int), map[String]Int, *Int), not a value expression")
	}
	return te.Type, nil
}

func (tc *TypeChecker) requireIntBuiltinArg(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan, i int, builtin string) error {
	argType, err := tc.inferBuiltinArgType(args, i, argSpans, callSpan)
	if err != nil {
		return err
	}
	if argType.Ident != ast.TypeInt {
		sp := spanForCallArg(argSpans, i, args, callSpan)
		return reportBodyf(sp, "builtin-call", "%s() argument %d must be Int, got %s", builtin, i+1, argType.Ident)
	}
	return nil
}

func (tc *TypeChecker) dispatchMake(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) < 1 {
		return nil, true, reportBodyf(callSpan, "builtin-call", "make() expects at least 1 argument, got 0")
	}
	typ, err := tc.inferMakeNewTypeArg(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	switch typ.Ident {
	case ast.TypeArray:
		if len(args) < 2 || len(args) > 3 {
			return nil, true, reportBodyf(callSpan, "builtin-call", "make(Array(T), len[, cap]) expects 2 or 3 arguments, got %d", len(args))
		}
		for i := 1; i < len(args); i++ {
			if err := tc.requireIntBuiltinArg(args, argSpans, callSpan, i, "make"); err != nil {
				return nil, true, err
			}
		}
		return []ast.TypeNode{typ}, true, nil
	case ast.TypeMap:
		if len(args) < 1 || len(args) > 2 {
			return nil, true, reportBodyf(callSpan, "builtin-call", "make(map[K]V[, hint]) expects 1 or 2 arguments, got %d", len(args))
		}
		if len(args) == 2 {
			if err := tc.requireIntBuiltinArg(args, argSpans, callSpan, 1, "make"); err != nil {
				return nil, true, err
			}
		}
		return []ast.TypeNode{typ}, true, nil
	case ast.TypeChannel:
		if len(args) < 1 || len(args) > 2 {
			return nil, true, reportBodyf(callSpan, "builtin-call", "make(chan T[, buffer]) expects 1 or 2 arguments, got %d", len(args))
		}
		if len(args) == 2 {
			if err := tc.requireIntBuiltinArg(args, argSpans, callSpan, 1, "make"); err != nil {
				return nil, true, err
			}
		}
		return []ast.TypeNode{typ}, true, nil
	default:
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, reportBodyf(sp, "builtin-call", "make() first argument must be Array(T), map[K]V, or chan T, got %s", typ.Ident)
	}
}

func (tc *TypeChecker) dispatchNew(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 1 {
		return nil, true, reportBodyf(callSpan, "builtin-call", "new(T) expects exactly 1 argument, got %d", len(args))
	}
	typ, err := tc.inferMakeNewTypeArg(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	return []ast.TypeNode{ast.NewPointerType(typ)}, true, nil
}
