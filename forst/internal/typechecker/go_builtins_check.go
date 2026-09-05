package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) checkBuiltinFunctionCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":   "checkBuiltinFunctionCall",
		"calledFn":   fn.Name,
		"argsLength": len(args),
	}).Debugf("Validating builtin function call")

	if ret, ok, err := tc.checkBuiltinConversionCall(fn, args, argSpans, callSpan); ok {
		return ret, err
	}
	if ret, ok, err := tc.checkBuiltinDispatchCall(fn, args, argSpans, callSpan); ok {
		return ret, err
	}
	return tc.checkBuiltinParamTypesCall(fn, args, argSpans, callSpan)
}

func (tc *TypeChecker) checkBuiltinConversionCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if fn.Name == "string" && len(args) == 1 {
		return tc.checkBuiltinStringCall(fn, args, argSpans, callSpan)
	}
	if fn.Name == "[]byte" && len(args) == 1 {
		return tc.checkBuiltinBytesCall(fn, args, argSpans, callSpan)
	}
	return nil, false, nil
}

func (tc *TypeChecker) checkBuiltinStringCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	argType, err := tc.inferExpressionType(args[0])
	if err != nil {
		return nil, true, err
	}
	sp := spanForCallArg(argSpans, 0, args, callSpan)
	if len(argType) != 1 {
		return nil, true, reportBodyf(sp, "builtin-call", "string() expects one argument")
	}
	switch argType[0].Ident {
	case ast.TypeInt, ast.TypeBool:
		return []ast.TypeNode{fn.ReturnType}, true, nil
	default:
		if isByteSliceType(argType[0]) {
			return []ast.TypeNode{fn.ReturnType}, true, nil
		}
		if argType[0].Ident == ast.TypeResult {
			if idx, ok := args[0].(ast.IndexExpressionNode); ok && tc.isMapIndexRValue(idx) {
				return nil, true, reportBodyf(sp, "builtin-call",
					"map lookup has type Result(V, Error); use `ensure x is Ok()` (or bind and handle the Result) before using string()")
			}
		}
		return nil, true, reportBodyf(sp, "builtin-call", "string() unsupported operand type %s", argType[0].String())
	}
}

func (tc *TypeChecker) checkBuiltinBytesCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	argType, err := tc.inferExpressionType(args[0])
	if err != nil {
		return nil, true, err
	}
	sp := spanForCallArg(argSpans, 0, args, callSpan)
	if len(argType) != 1 || argType[0].Ident != ast.TypeString {
		return nil, true, reportBodyf(sp, "builtin-call", "[]byte() expects a String argument")
	}
	return []ast.TypeNode{fn.ReturnType}, true, nil
}

func (tc *TypeChecker) checkBuiltinDispatchCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if fn.Package != "" || fn.Name == "string" {
		return nil, false, nil
	}
	ret, ok, err := tc.tryDispatchGoBuiltin(fn, args, argSpans, callSpan)
	if ok {
		return ret, true, err
	}
	if err != nil {
		return nil, true, err
	}
	if fn.CheckKind == BuiltinCheckDispatch {
		return nil, true, reportBodyf(callSpan, "builtin-call", "internal: missing tryDispatchGoBuiltin case for %q", fn.Name)
	}
	return nil, false, nil
}

func (tc *TypeChecker) checkBuiltinParamTypesCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, error) {
	if !fn.IsVarArgs && len(args) != len(fn.ParamTypes) {
		return nil, tc.builtinArityError(fn, args, argSpans, callSpan)
	}
	for i, arg := range args {
		if err := tc.checkBuiltinArgCompatible(fn, i, arg, args, argSpans, callSpan); err != nil {
			return nil, err
		}
	}
	return []ast.TypeNode{fn.ReturnType}, nil
}

func (tc *TypeChecker) builtinArityError(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) error {
	tc.log.WithFields(logrus.Fields{
		"function": "checkBuiltinFunctionCall",
	}).Errorf("%s() expects %d arguments, got %d", fn.Name, len(fn.ParamTypes), len(args))
	var sp ast.SourceSpan
	if len(args) > len(fn.ParamTypes) {
		sp = spanForCallArg(argSpans, len(fn.ParamTypes), args, callSpan)
	} else {
		sp = callSpan
	}
	if !sp.IsSet() && len(args) > 0 {
		sp = spanForCallArg(argSpans, 0, args, callSpan)
	}
	return reportBodyf(sp, "builtin-call", "%s() expects %d arguments, got %d", fn.Name, len(fn.ParamTypes), len(args))
}

func (tc *TypeChecker) checkBuiltinArgCompatible(fn BuiltinFunction, i int, arg ast.ExpressionNode, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) error {
	argType, err := tc.inferExpressionType(arg)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "checkBuiltinFunctionCall",
		}).Errorf("Error inferring type for argument %d: %v", i+1, err)
		return err
	}
	sp := spanForCallArg(argSpans, i, args, callSpan)
	if len(argType) != 1 {
		return reportBodyf(sp, "builtin-call", "%s() argument %d must have a single type", fn.Name, i+1)
	}
	expectedType := expectedBuiltinArgType(fn, i)
	if !tc.IsTypeCompatible(argType[0], expectedType) {
		return reportBodyf(sp, "builtin-call", "%s() argument %d must be of type %s, got %s",
			fn.Name, i+1, expectedType.Ident, argType[0].Ident)
	}
	return nil
}

func expectedBuiltinArgType(fn BuiltinFunction, i int) ast.TypeNode {
	if !fn.IsVarArgs {
		return fn.ParamTypes[i]
	}
	if fn.Package == "fmt" {
		switch fn.Name {
		case "Printf":
			if i == 0 {
				return fn.ParamTypes[0]
			}
			return ast.TypeNode{Ident: ast.TypeObject}
		case "Print", "Println":
			return ast.TypeNode{Ident: ast.TypeObject}
		}
	}
	return fn.ParamTypes[0]
}

// tryDispatchGoBuiltin applies Go-aligned rules for predeclared builtins (Package "").
// If it returns handled=false, the caller falls back to generic ParamTypes checking.
func (tc *TypeChecker) tryDispatchGoBuiltin(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	if ret, ok, err := tc.dispatchGoBuiltinSliceOps(fn.Name, args, argSpans, callSpan); ok {
		return ret, true, err
	}
	if ret, ok, err := tc.dispatchGoBuiltinMapOps(fn.Name, args, argSpans, callSpan); ok {
		return ret, true, err
	}
	return tc.dispatchGoBuiltinMiscOps(fn.Name, args, argSpans, callSpan)
}

func (tc *TypeChecker) dispatchGoBuiltinSliceOps(name string, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	switch name {
	case "len":
		return tc.dispatchLen(args, argSpans, callSpan)
	case "cap":
		return tc.dispatchCap(args, argSpans, callSpan)
	case "append":
		return tc.dispatchAppend(args, argSpans, callSpan)
	case "copy":
		return tc.dispatchCopy(args, argSpans, callSpan)
	default:
		return nil, false, nil
	}
}

func (tc *TypeChecker) dispatchGoBuiltinMapOps(name string, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	switch name {
	case "delete":
		return tc.dispatchDelete(args, argSpans, callSpan)
	case "clear":
		return tc.dispatchClear(args, argSpans, callSpan)
	default:
		return nil, false, nil
	}
}

func (tc *TypeChecker) dispatchGoBuiltinMiscOps(name string, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	switch name {
	case "close":
		return tc.dispatchClose(args, argSpans, callSpan)
	case "min", "max":
		return tc.dispatchMinMax(name, args, argSpans, callSpan)
	case "complex":
		return tc.dispatchComplex(args, argSpans, callSpan)
	case "real", "imag":
		return tc.dispatchRealImag(name, args, argSpans, callSpan)
	case "panic":
		return tc.dispatchPanic(args, argSpans, callSpan)
	case "print", "println":
		return tc.dispatchPrintLike(args, argSpans, callSpan)
	case "recover":
		return tc.dispatchRecover(args, callSpan)
	case "make":
		return tc.dispatchMake(args, argSpans, callSpan)
	case "new":
		return tc.dispatchNew(args, argSpans, callSpan)
	default:
		return nil, false, nil
	}
}
