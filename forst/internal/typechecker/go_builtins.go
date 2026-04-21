package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// IsTypeCompatible checks if a type is compatible with an expected type,
// taking into account subtypes and type guards
func (tc *TypeChecker) IsTypeCompatible(actual ast.TypeNode, expected ast.TypeNode) bool {
	if a, ok := tc.expandTypeDefBinaryIfNeeded(actual); ok {
		actual = a
	}
	if e, ok := tc.expandTypeDefBinaryIfNeeded(expected); ok {
		expected = e
	}

	tc.log.WithFields(logrus.Fields{
		"actual":   actual.Ident,
		"expected": expected.Ident,
		"function": "IsTypeCompatible",
	}).Debug("Checking type compatibility")

	// Same identifier: simple types match; generic built-ins must compare type parameters.
	if actual.Ident == expected.Ident {
		switch actual.Ident {
		case ast.TypeResult:
			if len(actual.TypeParams) != 2 || len(expected.TypeParams) != 2 {
				return false
			}
			return tc.IsTypeCompatible(actual.TypeParams[0], expected.TypeParams[0]) &&
				tc.IsTypeCompatible(actual.TypeParams[1], expected.TypeParams[1])
		case ast.TypeTuple:
			if len(actual.TypeParams) != len(expected.TypeParams) {
				return false
			}
			for i := range actual.TypeParams {
				if !tc.IsTypeCompatible(actual.TypeParams[i], expected.TypeParams[i]) {
					return false
				}
			}
			return true
		case ast.TypeUnion, ast.TypeIntersection:
			if len(actual.TypeParams) != len(expected.TypeParams) {
				return false
			}
			for i := range actual.TypeParams {
				if !tc.IsTypeCompatible(actual.TypeParams[i], expected.TypeParams[i]) {
					return false
				}
			}
			return true
		default:
			tc.log.WithFields(logrus.Fields{
				"actual":   actual.Ident,
				"expected": expected.Ident,
				"function": "IsTypeCompatible",
			}).Debug("Direct type match")
			return true
		}
	}

	// Union on the left: Union(A,B) <: T iff A <: T and B <: T.
	if actual.Ident == ast.TypeUnion && len(actual.TypeParams) > 0 {
		for _, m := range actual.TypeParams {
			if !tc.IsTypeCompatible(m, expected) {
				return false
			}
		}
		return true
	}

	// Intersection on the right: S <: A & B iff S <: A and S <: B.
	if expected.Ident == ast.TypeIntersection && len(expected.TypeParams) > 0 {
		for _, m := range expected.TypeParams {
			if !tc.IsTypeCompatible(actual, m) {
				return false
			}
		}
		return true
	}

	// Intersection on the left: A & B <: T iff A <: T and B <: T.
	if actual.Ident == ast.TypeIntersection && len(actual.TypeParams) > 0 {
		for _, m := range actual.TypeParams {
			if !tc.IsTypeCompatible(m, expected) {
				return false
			}
		}
		return true
	}

	// Union on the right: S <: Union(A,B) iff S <: A or S <: B.
	if expected.Ident == ast.TypeUnion && len(expected.TypeParams) > 0 {
		for _, m := range expected.TypeParams {
			if tc.IsTypeCompatible(actual, m) {
				return true
			}
		}
		return false
	}

	// RFC 02: nominal `error X { ... }` (`TypeDefErrorExpr`) is assignable to the built-in `Error` type.
	if expected.Ident == ast.TypeError {
		if def, ok := tc.Defs[actual.Ident].(ast.TypeDefNode); ok {
			if _, ok := def.Expr.(ast.TypeDefErrorExpr); ok {
				tc.log.WithFields(logrus.Fields{
					"actual":   actual.Ident,
					"expected": expected.Ident,
					"function": "IsTypeCompatible",
				}).Debug("Nominal error type assignable to built-in Error")
				return true
			}
		}
	}

	// Assigning to TypeObject mirrors Go assignability to interface{} / any (empty interface).
	if expected.Ident == ast.TypeObject && actual.Ident != ast.TypeVoid {
		tc.log.WithFields(logrus.Fields{
			"actual":   actual.Ident,
			"expected": expected.Ident,
			"function": "IsTypeCompatible",
		}).Debug("Actual type assignable to TypeObject")
		return true
	}

	// Value compatible with *T when compatible with T for non-scalar T (shape literals for *User,
	// etc.). Scalars still require an explicit pointer so String does not satisfy *String.
	if expected.Ident == ast.TypePointer && len(expected.TypeParams) == 1 {
		inner := expected.TypeParams[0]
		if !isScalarTypeIdent(inner.Ident) && tc.IsTypeCompatible(actual, inner) {
			tc.log.WithFields(logrus.Fields{
				"actual":   actual.Ident,
				"expected": expected.Ident,
				"function": "IsTypeCompatible",
			}).Debug("Actual type compatible with pointer element type")
			return true
		}
	}

	// Check if actual type is an alias of expected type
	actualDef, actualExists := tc.Defs[actual.Ident]
	if actualExists {
		if typeDef, ok := actualDef.(ast.TypeDefNode); ok {
			if typeDefExpr, ok := typeDefAssertionFromExpr(typeDef.Expr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil {
					baseType := ast.TypeNode{Ident: *typeDefExpr.Assertion.BaseType}
					if tc.IsTypeCompatible(baseType, expected) {
						tc.log.WithFields(logrus.Fields{
							"actual":   actual.Ident,
							"expected": expected.Ident,
							"function": "IsTypeCompatible",
						}).Debug("Actual type is alias of expected type")
						return true
					}
				}
			}
		}
	}

	// Check if expected type is an alias of actual type
	expectedDef, expectedExists := tc.Defs[expected.Ident]
	if expectedExists {
		if typeDef, ok := expectedDef.(ast.TypeDefNode); ok {
			if typeDefExpr, ok := typeDefAssertionFromExpr(typeDef.Expr); ok {
				if typeDefExpr.Assertion != nil && typeDefExpr.Assertion.BaseType != nil {
					baseType := ast.TypeNode{Ident: *typeDefExpr.Assertion.BaseType}
					if tc.IsTypeCompatible(actual, baseType) {
						tc.log.WithFields(logrus.Fields{
							"actual":   actual.Ident,
							"expected": expected.Ident,
							"function": "IsTypeCompatible",
						}).Debug("Expected type is alias of actual type")
						return true
					}
				}
			}
		}
	}

	// Check for structural compatibility between hash-based types and user-defined types
	if actualDef != nil && expectedDef != nil {
		tc.log.WithFields(logrus.Fields{
			"actual":   actual.Ident,
			"expected": expected.Ident,
			"function": "IsTypeCompatible",
		}).Info("Checking structural compatibility")

		actualShape, actualShapeOk := tc.getShapeFromTypeDef(actualDef)
		expectedShape, expectedShapeOk := tc.getShapeFromTypeDef(expectedDef)

		tc.log.WithFields(logrus.Fields{
			"actual":          actual.Ident,
			"expected":        expected.Ident,
			"actualShapeOk":   actualShapeOk,
			"expectedShapeOk": expectedShapeOk,
			"function":        "IsTypeCompatible",
		}).Info("Shape extraction results")

		if actualShapeOk && expectedShapeOk {
			// Prefer shapesHaveSameStructure: it resolves assertion fields and uses assignability for
			// mismatched type identifiers (e.g. inferred literal ctx vs AppContext).
			identical := tc.shapesHaveSameStructure(*actualShape, *expectedShape)
			tc.log.WithFields(logrus.Fields{
				"actual":    actual.Ident,
				"expected":  expected.Ident,
				"identical": identical,
				"function":  "IsTypeCompatible",
			}).Info("Structural compatibility check result")

			if identical {
				tc.log.WithFields(logrus.Fields{
					"actual":   actual.Ident,
					"expected": expected.Ident,
					"function": "IsTypeCompatible",
				}).Info("Shapes are structurally identical")
				return true
			}
		} else {
			tc.log.WithFields(logrus.Fields{
				"actual":          actual.Ident,
				"expected":        expected.Ident,
				"actualShapeOk":   actualShapeOk,
				"expectedShapeOk": expectedShapeOk,
				"function":        "IsTypeCompatible",
			}).Info("Could not extract shapes for structural comparison")
		}
	} else {
		tc.log.WithFields(logrus.Fields{
			"actual":      actual.Ident,
			"expected":    expected.Ident,
			"actualDef":   actualDef != nil,
			"expectedDef": expectedDef != nil,
			"function":    "IsTypeCompatible",
		}).Info("Skipping structural compatibility - missing type definitions")
	}

	if tc.shapeExpectationMatches(actual, expected) {
		return true
	}

	tc.log.WithFields(logrus.Fields{
		"actual":   actual.Ident,
		"expected": expected.Ident,
		"function": "IsTypeCompatible",
	}).Debug("Types are not compatible")
	return false
}

// shapeExpectationMatches handles hash-typed shape literals assigned to a named parameter type that
// has no Defs entry but was bound during inferShapeType (see shapeExpectations on TypeChecker).
func (tc *TypeChecker) shapeExpectationMatches(actual ast.TypeNode, expected ast.TypeNode) bool {
	if tc.shapeExpectations == nil || expected.Ident == "" {
		return false
	}
	expShape, ok := tc.shapeExpectations[expected.Ident]
	if !ok {
		return false
	}
	actualDef, ok := tc.Defs[actual.Ident]
	if !ok {
		return false
	}
	actualShape, ok := tc.getShapeFromTypeDef(actualDef)
	if !ok {
		return false
	}
	return tc.shapesHaveSameStructure(*actualShape, expShape)
}

func isScalarTypeIdent(id ast.TypeIdent) bool {
	switch id {
	case ast.TypeString, ast.TypeInt, ast.TypeFloat, ast.TypeBool:
		return true
	default:
		return false
	}
}

// getShapeFromTypeDef extracts the shape from a TypeDefNode if it is shape-backed (ordinary shape or error payload).
func (tc *TypeChecker) getShapeFromTypeDef(def ast.Node) (*ast.ShapeNode, bool) {
	if typeDef, ok := def.(ast.TypeDefNode); ok {
		return ast.PayloadShape(typeDef.Expr)
	}
	return nil, false
}

// shapesAreStructurallyIdentical returns true if two ShapeNodes have the same fields and types
func (tc *TypeChecker) shapesAreStructurallyIdentical(a, b ast.ShapeNode) bool {
	if len(a.Fields) != len(b.Fields) {
		return false
	}
	for name, fieldA := range a.Fields {
		fieldB, ok := b.Fields[name]
		if !ok {
			return false
		}
		// Compare field types (ignoring assertions for now)
		if fieldA.Type != nil && fieldB.Type != nil {
			// If either type is unknown (?), treat them as compatible
			if fieldA.Type.Ident == "?" || fieldB.Type.Ident == "?" {
				// Unknown types are compatible with any concrete type
				continue
			}
			if fieldA.Type.Ident == fieldB.Type.Ident {
				continue
			}
			// Hash-based structural types vs named shapes (e.g. inferred literal vs AppContext) must use
			// full assignability, not identifier equality only.
			if tc.IsTypeCompatible(*fieldA.Type, *fieldB.Type) {
				continue
			}
			return false
		} else if fieldA.Shape != nil && fieldB.Shape != nil {
			if !tc.shapesAreStructurallyIdentical(*fieldA.Shape, *fieldB.Shape) {
				return false
			}
		} else if (fieldA.Type != nil) != (fieldB.Type != nil) || (fieldA.Shape != nil) != (fieldB.Shape != nil) {
			return false
		}
	}
	return true
}

// checkBuiltinFunctionCall validates a call to a built-in function.
// argSpans and callSpan come from the parser when available; pass nil / zero value otherwise.
func (tc *TypeChecker) checkBuiltinFunctionCall(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":   "checkBuiltinFunctionCall",
		"calledFn":   fn.Name,
		"argsLength": len(args),
	}).Debugf("Validating builtin function call")

	if fn.Name == "string" && len(args) == 1 {
		argType, err := tc.inferExpressionType(args[0])
		if err != nil {
			return nil, err
		}
		sp := spanForCallArg(argSpans, 0, args, callSpan)
		if len(argType) != 1 {
			return nil, diagnosticf(sp, "builtin-call", "string() expects one argument")
		}
		switch argType[0].Ident {
		case ast.TypeInt:
			return []ast.TypeNode{fn.ReturnType}, nil
		default:
			if argType[0].Ident == ast.TypeResult {
				if idx, ok := args[0].(ast.IndexExpressionNode); ok && tc.isMapIndexRValue(idx) {
					return nil, diagnosticf(sp, "builtin-call",
						"map lookup has type Result(V, Error); use `ensure x is Ok()` (or bind and handle the Result) before using string()")
				}
			}
			return nil, diagnosticf(sp, "builtin-call", "string() unsupported operand type %s", argType[0].Ident)
		}
	}

	// Go predeclared builtins (empty package): semantics are implemented in tryDispatchGoBuiltin, not ParamTypes.
	if fn.Package == "" && fn.Name != "string" {
		ret, ok, err := tc.tryDispatchGoBuiltin(fn, args, argSpans, callSpan)
		if ok {
			return ret, err
		}
		if err != nil {
			return nil, err
		}
		if fn.CheckKind == BuiltinCheckDispatch {
			return nil, diagnosticf(callSpan, "builtin-call", "internal: missing tryDispatchGoBuiltin case for %q", fn.Name)
		}
	}

	// Check argument count
	if !fn.IsVarArgs && len(args) != len(fn.ParamTypes) {
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
		return nil, diagnosticf(sp, "builtin-call", "%s() expects %d arguments, got %d", fn.Name, len(fn.ParamTypes), len(args))
	}

	// Check argument types
	for i, arg := range args {
		argType, err := tc.inferExpressionType(arg)
		if err != nil {
			tc.log.WithFields(logrus.Fields{
				"function": "checkBuiltinFunctionCall",
			}).Errorf("Error inferring type for argument %d: %v", i+1, err)
			return nil, err
		}
		sp := spanForCallArg(argSpans, i, args, callSpan)
		tc.log.WithFields(logrus.Fields{
			"function": "checkBuiltinFunctionCall",
		}).Debugf("Arg %d inferred type: %+v", i+1, argType)
		if len(argType) != 1 {
			tc.log.WithFields(logrus.Fields{
				"function": "checkBuiltinFunctionCall",
			}).Errorf("%s() argument %d must have a single type, got %d", fn.Name, i+1, len(argType))
			return nil, diagnosticf(sp, "builtin-call", "%s() argument %d must have a single type", fn.Name, i+1)
		}

		var expectedType ast.TypeNode
		if !fn.IsVarArgs {
			expectedType = fn.ParamTypes[i]
		} else if fn.Package == "fmt" {
			switch fn.Name {
			case "Printf":
				if i == 0 {
					expectedType = fn.ParamTypes[0] // format: String
				} else {
					expectedType = ast.TypeNode{Ident: ast.TypeObject}
				}
			case "Print", "Println":
				expectedType = ast.TypeNode{Ident: ast.TypeObject}
			default:
				expectedType = fn.ParamTypes[0]
			}
		} else {
			// For other varargs, all args must match the first param type.
			expectedType = fn.ParamTypes[0]
		}
		tc.log.WithFields(logrus.Fields{
			"function": "checkBuiltinFunctionCall",
		}).Debugf("Arg %d expected type: %+v", i+1, expectedType)

		if !tc.IsTypeCompatible(argType[0], expectedType) {
			tc.log.WithFields(logrus.Fields{
				"function": "checkBuiltinFunctionCall",
			}).Errorf("%s() argument %d must be of type %s, got %s", fn.Name, i+1, expectedType.Ident, argType[0].Ident)
			return nil, diagnosticf(sp, "builtin-call", "%s() argument %d must be of type %s, got %s",
				fn.Name, i+1, expectedType.Ident, argType[0].Ident)
		}
	}

	return []ast.TypeNode{fn.ReturnType}, nil
}

func (tc *TypeChecker) inferBuiltinArgType(args []ast.ExpressionNode, i int, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) (ast.TypeNode, error) {
	if i < 0 || i >= len(args) {
		return ast.TypeNode{}, diagnosticf(callSpan, "builtin-call", "internal: missing argument %d", i+1)
	}
	sp := spanForCallArg(argSpans, i, args, callSpan)
	ts, err := tc.inferExpressionType(args[i])
	if err != nil {
		return ast.TypeNode{}, err
	}
	if len(ts) != 1 {
		return ast.TypeNode{}, diagnosticf(sp, "builtin-call", "argument %d must have a single type", i+1)
	}
	return ts[0], nil
}

// tryDispatchGoBuiltin applies Go-aligned rules for predeclared builtins (Package "").
// If it returns handled=false, the caller falls back to generic ParamTypes checking.
func (tc *TypeChecker) tryDispatchGoBuiltin(fn BuiltinFunction, args []ast.ExpressionNode, argSpans []ast.SourceSpan, callSpan ast.SourceSpan) ([]ast.TypeNode, bool, error) {
	switch fn.Name {
	case "len":
		return tc.dispatchLen(args, argSpans, callSpan)
	case "cap":
		return tc.dispatchCap(args, argSpans, callSpan)
	case "append":
		return tc.dispatchAppend(args, argSpans, callSpan)
	case "copy":
		return tc.dispatchCopy(args, argSpans, callSpan)
	case "delete":
		return tc.dispatchDelete(args, argSpans, callSpan)
	case "close":
		return tc.dispatchClose(args, argSpans, callSpan)
	case "clear":
		return tc.dispatchClear(args, argSpans, callSpan)
	case "min", "max":
		return tc.dispatchMinMax(fn.Name, args, argSpans, callSpan)
	case "complex":
		return tc.dispatchComplex(args, argSpans, callSpan)
	case "real", "imag":
		return tc.dispatchRealImag(fn.Name, args, argSpans, callSpan)
	case "panic":
		return tc.dispatchPanic(args, argSpans, callSpan)
	case "print", "println":
		return tc.dispatchPrintLike(args, argSpans, callSpan)
	case "recover":
		return tc.dispatchRecover(args, callSpan)
	case "make", "new":
		return nil, true, diagnosticf(callSpan, "builtin-call",
			"%s() with a type argument is not supported yet: use Forst type syntax (e.g. Array(Int)) when the parser accepts types in call position, or use a small Go helper via import", fn.Name)

	default:
		return nil, false, nil
	}
}

// isMapIndexRValue is true when expr is a subscript whose target is a map type (rvalue read).
func (tc *TypeChecker) isMapIndexRValue(expr ast.IndexExpressionNode) bool {
	tts, err := tc.inferExpressionType(expr.Target)
	if err != nil || len(tts) != 1 {
		return false
	}
	return tts[0].Ident == ast.TypeMap && len(tts[0].TypeParams) >= 2
}
