package typechecker

import "forst/internal/ast"

func (tc *TypeChecker) dispatchLen(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 1 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "len() expects 1 argument, got %d", len(args))
	}
	argType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if !lenOperandAllowed(argType) {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "len() invalid operand type %s", argType.Ident)
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)}, true, nil
}

func (tc *TypeChecker) dispatchCap(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 1 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "cap() expects 1 argument, got %d", len(args))
	}
	argType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if !capOperandAllowed(argType) {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "cap() invalid operand type %s", argType.Ident)
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)}, true, nil
}

func (tc *TypeChecker) dispatchAppend(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) < 2 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "append() expects at least 2 arguments, got %d", len(args))
	}
	sliceType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if sliceType.Ident != ast.TypeArray {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "append() first argument must be a slice, got %s", sliceType.Ident)
	}
	elemType, ok := sliceElementType(sliceType)
	if !ok {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "append() slice must have an element type")
	}
	for i := 1; i < len(args); i++ {
		argType, err := tc.inferBuiltinArgType(args, i, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if !tc.IsTypeCompatible(argType, elemType) {
			sp := spanForCallArg(argSpans, i, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "append() argument %d must be assignable to slice element %s, got %s",
				i+1, elemType.Ident, argType.Ident)
		}
	}
	return []ast.TypeNode{sliceType}, true, nil
}

func (tc *TypeChecker) dispatchCopy(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 2 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "copy() expects 2 arguments, got %d", len(args))
	}
	dstType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	srcType, err := tc.inferBuiltinArgType(args, 1, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if dstType.Ident == ast.TypeArray && len(dstType.TypeParams) == 1 && dstType.TypeParams[0].Ident == ast.TypeInt && srcType.Ident == ast.TypeString {
		return []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)}, true, nil
	}
	if dstType.Ident != ast.TypeArray || srcType.Ident != ast.TypeArray {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "copy() expects two slices, or []Int with String (Go copy to []byte)")
	}
	dstElem, okDst := sliceElementType(dstType)
	srcElem, okSrc := sliceElementType(srcType)
	if !okDst || !okSrc {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "copy() could not read slice element types")
	}
	if !builtinTypesIdenticalOrdered(dstElem, srcElem) {
		sp := spanForCallArg(argSpans, 1, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "copy() slice element types must match")
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)}, true, nil
}

func (tc *TypeChecker) dispatchDelete(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 2 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "delete() expects 2 arguments, got %d", len(args))
	}
	mapType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	keyType, err := tc.inferBuiltinArgType(args, 1, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	mapKeyType, _, ok := mapKeyValueTypes(mapType)
	if !ok {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "delete() first argument must be a map, got %s", mapType.Ident)
	}
	if !tc.IsTypeCompatible(keyType, mapKeyType) {
		sp := spanForCallArg(argSpans, 1, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "delete() key type incompatible with map key %s", mapKeyType.Ident)
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeVoid)}, true, nil
}

func (tc *TypeChecker) dispatchClose(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 1 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "close() expects 1 argument, got %d", len(args))
	}
	if _, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan); err != nil {
		return nil, true, err
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeVoid)}, true, nil
}

func (tc *TypeChecker) dispatchClear(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 1 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "clear() expects 1 argument, got %d", len(args))
	}
	argType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if !clearOperandAllowed(argType) {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "clear() expects a map or slice, got %s", argType.Ident)
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeVoid)}, true, nil
}

func (tc *TypeChecker) dispatchMinMax(functionName string, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) < 1 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "%s() expects at least 1 argument", functionName)
	}
	firstType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if !isOrderedBuiltinType(firstType) {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "%s() expects ordered types (Int, Float, String), got %s", functionName, firstType.Ident)
	}
	for i := 1; i < len(args); i++ {
		argType, err := tc.inferBuiltinArgType(args, i, argSpans, callSpan)
		if err != nil {
			return nil, true, err
		}
		if !builtinTypesIdenticalOrdered(firstType, argType) {
			sp := spanForCallArg(argSpans, i, args, callSpan)
			return nil, true, diagnosticf(sp, "builtin-call", "%s() arguments must have the same type", functionName)
		}
	}
	return []ast.TypeNode{firstType}, true, nil
}

func (tc *TypeChecker) dispatchComplex(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 2 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "complex() expects 2 arguments, got %d", len(args))
	}
	leftType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	rightType, err := tc.inferBuiltinArgType(args, 1, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if leftType.Ident != ast.TypeFloat || rightType.Ident != ast.TypeFloat {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "complex() expects Float arguments, got %s and %s", leftType.Ident, rightType.Ident)
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeObject)}, true, nil
}

func (tc *TypeChecker) dispatchRealImag(functionName string, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 1 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "%s() expects 1 argument, got %d", functionName, len(args))
	}
	argType, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan)
	if err != nil {
		return nil, true, err
	}
	if argType.Ident != ast.TypeObject {
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		return nil, true, diagnosticf(sp, "builtin-call", "%s() expects a complex value, got %s", functionName, argType.Ident)
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeFloat)}, true, nil
}

func (tc *TypeChecker) dispatchPanic(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 1 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "panic() expects 1 argument, got %d", len(args))
	}
	if _, err := tc.inferBuiltinArgType(args, 0, argSpans, callSpan); err != nil {
		return nil, true, err
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeVoid)}, true, nil
}

func (tc *TypeChecker) dispatchPrintLike(args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	for i := range args {
		if _, err := tc.inferBuiltinArgType(args, i, argSpans, callSpan); err != nil {
			return nil, true, err
		}
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeVoid)}, true, nil
}

func (tc *TypeChecker) dispatchRecover(args []ast.ExpressionNode, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if len(args) != 0 {
		return nil, true, diagnosticf(callSpan, "builtin-call", "recover() expects 0 arguments, got %d", len(args))
	}
	return []ast.TypeNode{ast.NewBuiltinType(ast.TypeObject)}, true, nil
}
