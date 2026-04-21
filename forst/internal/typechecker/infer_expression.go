package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"strings"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferExpressionType(expr ast.Node) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function": "inferExpressionType",
		"expr":     expr,
	}).Debugf("Starting type inference for expression")
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Left, e.Right, e.Operator)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, nil

	case ast.UnaryExpressionNode:
		inferredType, err := tc.unifyTypes(e.Operand, nil, e.Operator)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, nil

	case ast.IntLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeInt}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.FloatLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeFloat}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.StringLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeString}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.BoolLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeBool}
		tc.storeInferredType(e, []ast.TypeNode{typ})
		return []ast.TypeNode{typ}, nil

	case ast.ArrayLiteralNode:
		if len(e.Value) == 0 {
			elem := ast.TypeNode{Ident: ast.TypeInt}
			if e.Type.Ident != ast.TypeImplicit && e.Type.Ident != "" {
				elem = e.Type
			}
			arr := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elem}}
			tc.storeInferredType(e, []ast.TypeNode{arr})
			return []ast.TypeNode{arr}, nil
		}
		var elemType ast.TypeNode
		for i, el := range e.Value {
			ts, err := tc.inferExpressionType(el)
			if err != nil {
				return nil, err
			}
			if len(ts) != 1 {
				return nil, fmt.Errorf("array element %d: expected a single type", i)
			}
			if i == 0 {
				elemType = ts[0]
			} else if elemType.Ident != ts[0].Ident {
				return nil, fmt.Errorf("array literal: mixed element types %s and %s", elemType.Ident, ts[0].Ident)
			}
		}
		arr := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elemType}}
		tc.storeInferredType(e, []ast.TypeNode{arr})
		return []ast.TypeNode{arr}, nil

	case ast.VariableNode:
		// Look up the variable's type and store it for this node. Flow-sensitive facts and FFI
		// invalidation (future) belong in FlowTypeFact / a separate layer — not in Meet/Join.
		typ, narrowGuards, predDisplay, err := tc.lookupVariableForExpression(&e, tc.CurrentScope())
		if err != nil {
			return nil, err
		}
		tc.log.Tracef("Variable type: %+v, node: %+v, type params: %+v, (original: %+v of type %T)", typ, e, typ.TypeParams, e, e)
		tc.storeInferredType(e, []ast.TypeNode{typ})
		if e.Ident.Span.IsSet() {
			k := variableOccurrenceKey{ident: e.Ident.ID, span: e.Ident.Span}
			if len(narrowGuards) > 0 {
				tc.variableOccurrenceNarrowingGuards[k] = append([]string(nil), narrowGuards...)
			}
			if predDisplay != "" {
				tc.variableOccurrenceNarrowingPredicateDisplay[k] = predDisplay
			}
		}
		return []ast.TypeNode{typ}, nil

	case ast.IndexExpressionNode:
		targetTypes, err := tc.inferExpressionType(e.Target)
		if err != nil {
			return nil, err
		}
		if len(targetTypes) != 1 {
			return nil, fmt.Errorf("index expression: target must have a single type, got %d", len(targetTypes))
		}
		t := targetTypes[0]
		indexTypes, err := tc.inferExpressionType(e.Index)
		if err != nil {
			return nil, err
		}
		if len(indexTypes) != 1 {
			return nil, fmt.Errorf("index expression: index must have a single type")
		}
		if t.Ident == ast.TypeMap && len(t.TypeParams) >= 2 {
			wantK, wantV := t.TypeParams[0], t.TypeParams[1]
			if !tc.IsTypeCompatible(indexTypes[0], wantK) {
				return nil, fmt.Errorf("map index: key type want %s, got %s", wantK.Ident, indexTypes[0].Ident)
			}
			// Rvalue map lookup is Result(V, Error): present key → Ok(value); missing key → Err.
			resultType := ast.TypeNode{
				Ident: ast.TypeResult,
				TypeParams: []ast.TypeNode{
					wantV,
					{Ident: ast.TypeError},
				},
			}
			tc.storeInferredType(e, []ast.TypeNode{resultType})
			return []ast.TypeNode{resultType}, nil
		}
		if t.Ident != ast.TypeArray || len(t.TypeParams) < 1 {
			return nil, fmt.Errorf("index expression: target must be a map, slice, or array, got %s", t.Ident)
		}
		if indexTypes[0].Ident != ast.TypeInt {
			return nil, fmt.Errorf("index expression: slice/array index must be Int, got %s", indexTypes[0].Ident)
		}
		elem := t.TypeParams[0]
		tc.storeInferredType(e, []ast.TypeNode{elem})
		return []ast.TypeNode{elem}, nil

	case ast.FunctionCallNode:
		tc.log.WithFields(logrus.Fields{
			"function": "inferExpressionType",
			"expr":     expr,
		}).Tracef("Checking function call: %s with %d arguments", e.Function.ID, len(e.Arguments))

		var argTypes [][]ast.TypeNode
		if sig, ok := tc.Functions[e.Function.ID]; ok && len(e.Arguments) == len(sig.Parameters) {
			argTypes = make([][]ast.TypeNode, len(e.Arguments))
			for i, arg := range e.Arguments {
				exp := &sig.Parameters[i].Type
				ts, err := tc.inferExpressionTypeWithExpected(arg, exp)
				if err != nil {
					return nil, err
				}
				argTypes[i] = ts
			}
		} else {
			argTypes = make([][]ast.TypeNode, 0, len(e.Arguments))
			for _, arg := range e.Arguments {
				ts, err := tc.inferExpressionType(arg)
				if err != nil {
					return nil, err
				}
				argTypes = append(argTypes, ts)
			}
		}

		if signature, exists := tc.Functions[e.Function.ID]; exists {
			tc.log.WithFields(logrus.Fields{
				"function": "inferExpressionType",
				"expr":     expr,
			}).Tracef("Found function signature for %s: %v", e.Function.ID, signature.ReturnTypes)
			if len(argTypes) != len(signature.Parameters) {
				var sp ast.SourceSpan
				if len(argTypes) > len(signature.Parameters) {
					sp = spanForCallArg(e.ArgSpans, len(signature.Parameters), e.Arguments, e.CallSpan)
				} else {
					sp = e.CallSpan
				}
				if !sp.IsSet() {
					sp = e.Function.Span
				}
				return nil, diagnosticf(sp, "call-arity", "function %s expects %d arguments, got %d",
					e.Function.ID, len(signature.Parameters), len(argTypes))
			}
			for i, param := range signature.Parameters {
				sp := spanForCallArg(e.ArgSpans, i, e.Arguments, e.CallSpan)
				if len(argTypes[i]) != 1 {
					return nil, diagnosticf(sp, "call-type", "argument %d to %s must have a single type, got %d",
						i+1, e.Function.ID, len(argTypes[i]))
				}
				if !tc.IsTypeCompatible(argTypes[i][0], param.Type) {
					return nil, diagnosticf(sp, "call-type", "argument %d to %s: expected type %s, got %s",
						i+1, e.Function.ID, param.Type.Ident, argTypes[i][0].Ident)
				}
			}
			tc.storeInferredType(e, signature.ReturnTypes)
			return signature.ReturnTypes, nil
		}

		// For type guards, we need to ensure they return boolean
		if typeGuard, exists := tc.scopeStack.globalScope().Symbols[e.Function.ID]; exists && typeGuard.Kind == SymbolTypeGuard {
			tc.log.WithFields(logrus.Fields{
				"function": "inferExpressionType",
				"expr":     expr,
			}).Tracef("Found type guard %s with types: %v", e.Function.ID, typeGuard.Types)
			// Type guards return boolean when called
			return []ast.TypeNode{{Ident: ast.TypeBool}}, nil
		}

		// First check if this is a local variable or method call
		if varType, exists := tc.scopeStack.LookupVariableType(e.Function.ID); exists {
			tc.log.WithFields(logrus.Fields{
				"function": "inferExpressionType",
				"expr":     expr,
			}).Tracef("Found local variable %s with type: %v", e.Function.ID, varType)
			return varType, nil
		}

		// Then check if this is a package-qualified function call
		parts := strings.Split(string(e.Function.ID), ".")
		if len(parts) == 2 {
			pkgName := parts[0]
			funcName := parts[1]

			// First check if pkgName is a local variable
			if varType, exists := tc.scopeStack.LookupVariableType(ast.Identifier(pkgName)); exists {
				tc.log.WithFields(logrus.Fields{
					"function": "inferExpressionType",
					"expr":     expr,
				}).Tracef("Found local variable %s with type: %v", pkgName, varType)
				// Check if the method is valid for this type
				returnType, err := tc.inferMethodCallType(ast.Identifier(pkgName), varType, funcName, e, argTypes)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, returnType)
				return returnType, nil
			}

			// Imported Go package (Forst↔Go boundary): batch or lazy go/packages load
			if gp := tc.goPackageForImportLocal(pkgName); gp != nil {
				ret, err := tc.checkGoQualifiedCall(gp, pkgName, funcName, e, argTypes, true)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, ret)
				return ret, nil
			}

			// If not a local variable, check for built-in functions
			qualifiedName := pkgName + "." + funcName
			if builtin, exists := BuiltinFunctions[qualifiedName]; exists {
				tc.log.WithFields(logrus.Fields{
					"function": "inferExpressionType",
					"expr":     expr,
				}).Tracef("Found built-in function %s", qualifiedName)
				returnType, err := tc.checkBuiltinFunctionCall(builtin, e.Arguments, e.ArgSpans, e.CallSpan)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, returnType)
				return returnType, nil
			}
		} else {
			// Check for unqualified built-in functions (like len)
			if builtin, exists := BuiltinFunctions[string(e.Function.ID)]; exists {
				tc.log.WithFields(logrus.Fields{
					"function": "inferExpressionType",
					"expr":     expr,
				}).Tracef("Found built-in function %s", e.Function.ID)
				returnType, err := tc.checkBuiltinFunctionCall(builtin, e.Arguments, e.ArgSpans, e.CallSpan)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, returnType)
				return returnType, nil
			}
			// Dot-imported Go package: import . "strings" → NewReader(...)
			spDot := e.Function.Span
			if !spDot.IsSet() {
				spDot = e.CallSpan
			}
			if gp, err := tc.lookupDotImportFunc(string(e.Function.ID), spDot); err != nil {
				return nil, err
			} else if gp != nil {
				ret, err := tc.checkGoQualifiedCall(gp, gp.Path(), string(e.Function.ID), e, argTypes, true)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, ret)
				return ret, nil
			}
		}

		tc.log.WithFields(logrus.Fields{
			"function": "inferExpressionType",
			"expr":     expr,
		}).Tracef("No function found for %s", e.Function.ID)
		sp := e.Function.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, diagnosticf(sp, "undefined-identifier", "unknown identifier: %s", e.Function.ID)

	case ast.ShapeNode:
		inferredType, err := tc.inferShapeType(e, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to infer shape type: %w", err)
		}
		tc.storeInferredType(e, []ast.TypeNode{inferredType})
		return []ast.TypeNode{inferredType}, nil

	case ast.AssertionNode:
		inferredType, err := tc.InferAssertionType(&e, false, "", nil)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, inferredType)
		return inferredType, nil

	case ast.ReferenceNode:
		valueType, err := tc.inferExpressionType(e.Value)
		if err != nil {
			return nil, err
		}
		referenceType := ast.TypeNode{
			Ident:      ast.TypePointer,
			TypeParams: valueType,
		}
		tc.storeInferredType(e, []ast.TypeNode{referenceType})
		return []ast.TypeNode{referenceType}, nil

	case ast.DereferenceNode:
		valueType, err := tc.inferExpressionType(e.Value)
		if err != nil {
			return nil, err
		}
		if len(valueType) != 1 {
			return nil, fmt.Errorf("dereference is only valid on single types, got %s", formatTypeList(valueType))
		}
		tc.log.Tracef("Dereference type identifier: %+v", valueType[0].Node)
		if valueType[0].Ident != ast.TypePointer {
			return nil, fmt.Errorf("dereference is only valid on pointer types, got %s", valueType[0].Ident)
		}
		tc.log.Tracef("Dereference type: %+v", valueType[0].TypeParams)
		tc.storeInferredType(e, valueType[0].TypeParams)
		return valueType[0].TypeParams, nil

	case ast.NilLiteralNode:
		// Return a special marker (empty slice) to indicate untyped nil; context must resolve
		return nil, nil

	case ast.MapLiteralNode:
		if e.Type.Ident != ast.TypeMap || len(e.Type.TypeParams) != 2 {
			return nil, fmt.Errorf("map literal: invalid type %v", e.Type)
		}
		wantK, wantV := e.Type.TypeParams[0], e.Type.TypeParams[1]
		for i, ent := range e.Entries {
			kt, err := tc.inferExpressionType(ent.Key)
			if err != nil {
				return nil, fmt.Errorf("map literal entry %d key: %w", i, err)
			}
			if len(kt) != 1 {
				return nil, fmt.Errorf("map literal entry %d key: expected one type", i)
			}
			if !tc.IsTypeCompatible(kt[0], wantK) {
				return nil, fmt.Errorf("map literal entry %d key: want %s, got %s", i, wantK.Ident, kt[0].Ident)
			}
			vt, err := tc.inferExpressionType(ent.Value)
			if err != nil {
				return nil, fmt.Errorf("map literal entry %d value: %w", i, err)
			}
			if len(vt) != 1 {
				return nil, fmt.Errorf("map literal entry %d value: expected one type", i)
			}
			if !tc.IsTypeCompatible(vt[0], wantV) {
				return nil, fmt.Errorf("map literal entry %d value: want %s, got %s", i, wantV.Ident, vt[0].Ident)
			}
		}
		tc.storeInferredType(e, []ast.TypeNode{e.Type})
		return []ast.TypeNode{e.Type}, nil

	case ast.OkExprNode:
		return nil, fmt.Errorf("Ok(...) is not a value constructor; use `is Ok()` / `ensure ... is Ok()` guards, or return a plain success value of type S for Result(S, F)")

	case ast.ErrExprNode:
		return nil, fmt.Errorf("Err(...) is not a value constructor; use `is Err()` / `ensure ...` and FFI/interop for failure values")

	default:
		tc.log.Tracef("Unhandled expression type: %T", expr)
		return nil, fmt.Errorf("cannot infer type for expression: %T", expr)
	}
}

// Checks if a method call is valid for a given type and returns its return type
func (tc *TypeChecker) inferMethodCallType(receiver ast.Identifier, varType []ast.TypeNode, methodName string, e ast.FunctionCallNode, argTypes [][]ast.TypeNode) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":   "inferMethodCallType",
		"varType":    varType,
		"methodName": methodName,
		"receiver":   receiver,
	}).Tracef("inferMethodCallType")

	if len(varType) != 1 {
		tc.log.WithFields(logrus.Fields{
			"function": "inferMethodCallType",
			"varType":  varType,
		}).Tracef("Method calls are only valid on single types, got %s", formatTypeList(varType))
		return nil, fmt.Errorf("method calls are only valid on single types, got %s", formatTypeList(varType))
	}

	t := varType[0]
	if t.IsResultType() && len(t.TypeParams) >= 2 {
		switch methodName {
		case "Ok":
			return []ast.TypeNode{t.TypeParams[0]}, nil
		case "Err":
			return []ast.TypeNode{t.TypeParams[1]}, nil
		default:
			return nil, fmt.Errorf("method %s() is not valid on type %s", methodName, t.String())
		}
	}

	if goRecv, ok := tc.variableGoTypes[receiver]; ok && goRecv != nil {
		ret, err := tc.checkGoMethodCall(goRecv, methodName, e, argTypes, true)
		if err != nil {
			return nil, err
		}
		tc.log.WithFields(logrus.Fields{
			"function":   "inferMethodCallType",
			"receiver":   receiver,
			"methodName": methodName,
		}).Tracef("Go method call: %v", ret)
		return ret, nil
	}

	// *T method calls: lower to element type for built-in / opaque Go receivers.
	if t.Ident == ast.TypePointer && len(t.TypeParams) == 1 {
		t = t.TypeParams[0]
	}

	returnType, err := tc.CheckBuiltinMethod(t, methodName, e.Arguments)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function":   "inferMethodCallType",
			"varType":    varType,
			"methodName": methodName,
		}).Tracef("Error checking built-in method: %v", err)
		return nil, err
	}

	tc.log.WithFields(logrus.Fields{
		"function":   "inferMethodCallType",
		"varType":    varType,
		"methodName": methodName,
	}).Tracef("Successfully inferred method call type: %v", returnType)
	return returnType, nil
}

// inferIndexExpressionAsAssignTarget types an index expression as an assignment target (m[k] = x or xs[i] = x).
// Map cells use element type V; rvalue map reads elsewhere use Result(V, Error) via inferExpressionType.
func (tc *TypeChecker) inferIndexExpressionAsAssignTarget(e ast.IndexExpressionNode) ([]ast.TypeNode, error) {
	targetTypes, err := tc.inferExpressionType(e.Target)
	if err != nil {
		return nil, err
	}
	if len(targetTypes) != 1 {
		return nil, fmt.Errorf("index expression: target must have a single type, got %d", len(targetTypes))
	}
	t := targetTypes[0]
	indexTypes, err := tc.inferExpressionType(e.Index)
	if err != nil {
		return nil, err
	}
	if len(indexTypes) != 1 {
		return nil, fmt.Errorf("index expression: index must have a single type")
	}
	if t.Ident == ast.TypeMap && len(t.TypeParams) >= 2 {
		wantK, wantV := t.TypeParams[0], t.TypeParams[1]
		if !tc.IsTypeCompatible(indexTypes[0], wantK) {
			return nil, fmt.Errorf("map index: key type want %s, got %s", wantK.Ident, indexTypes[0].Ident)
		}
		tc.storeInferredType(e, []ast.TypeNode{wantV})
		return []ast.TypeNode{wantV}, nil
	}
	if t.Ident != ast.TypeArray || len(t.TypeParams) < 1 {
		return nil, fmt.Errorf("index expression: target must be a map, slice, or array, got %s", t.Ident)
	}
	if indexTypes[0].Ident != ast.TypeInt {
		return nil, fmt.Errorf("index expression: slice/array index must be Int, got %s", indexTypes[0].Ident)
	}
	elem := t.TypeParams[0]
	tc.storeInferredType(e, []ast.TypeNode{elem})
	return []ast.TypeNode{elem}, nil
}

// inferExpressionTypeWithExpected infers an expression's type. For shape literals it passes the
// callee parameter type into inferShapeType so fields match the formal parameter (e.g. *String for
// `sessionId` when the parameter is AppMutation.Input(...)).
func (tc *TypeChecker) inferExpressionTypeWithExpected(expr ast.Node, expected *ast.TypeNode) ([]ast.TypeNode, error) {
	if expected != nil {
		switch x := expr.(type) {
		case ast.ShapeNode:
			inferredType, err := tc.inferShapeType(x, expected)
			if err != nil {
				return nil, err
			}
			tc.storeInferredType(expr, []ast.TypeNode{inferredType})
			return []ast.TypeNode{inferredType}, nil
		case *ast.ShapeNode:
			inferredType, err := tc.inferShapeType(*x, expected)
			if err != nil {
				return nil, err
			}
			tc.storeInferredType(expr, []ast.TypeNode{inferredType})
			return []ast.TypeNode{inferredType}, nil
		case ast.IndexExpressionNode:
			// Extension point: map/slice index with an expected type (e.g. generic calls). Today
			// expected is ignored; index inference is unchanged from inferExpressionType.
			return tc.inferExpressionType(x)
		}
	}
	return tc.inferExpressionType(expr)
}
