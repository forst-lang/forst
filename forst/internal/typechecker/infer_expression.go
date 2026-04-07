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
		if t.Ident != ast.TypeArray || len(t.TypeParams) < 1 {
			return nil, fmt.Errorf("index expression: target must be a slice or array, got %s", t.Ident)
		}
		indexTypes, err := tc.inferExpressionType(e.Index)
		if err != nil {
			return nil, err
		}
		if len(indexTypes) != 1 {
			return nil, fmt.Errorf("index expression: index must have a single type")
		}
		if indexTypes[0].Ident != ast.TypeInt {
			return nil, fmt.Errorf("index expression: index must be Int, got %s", indexTypes[0].Ident)
		}
		elem := t.TypeParams[0]
		tc.storeInferredType(e, []ast.TypeNode{elem})
		return []ast.TypeNode{elem}, nil

	case ast.FunctionCallNode:
		tc.log.WithFields(logrus.Fields{
			"function": "inferExpressionType",
			"expr":     expr,
		}).Tracef("Checking function call: %s with %d arguments", e.Function.ID, len(e.Arguments))

		argTypes := make([][]ast.TypeNode, 0, len(e.Arguments))
		for _, arg := range e.Arguments {
			ts, err := tc.inferExpressionType(arg)
			if err != nil {
				return nil, err
			}
			argTypes = append(argTypes, ts)
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
				returnType, err := tc.inferMethodCallType(varType, funcName, e.Arguments)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, returnType)
				return returnType, nil
			}

			// Imported Go package (Forst↔Go boundary) when go/packages load succeeded
			if tc.goPkgsByLocal != nil {
				if gp := tc.goPkgsByLocal[pkgName]; gp != nil {
					ret, err := tc.checkGoQualifiedCall(gp, pkgName, funcName, e, argTypes, true)
					if err != nil {
						return nil, err
					}
					tc.storeInferredType(e, ret)
					return ret, nil
				}
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
		ts, err := tc.inferExpressionType(e.Value)
		if err != nil {
			return nil, err
		}
		if len(ts) != 1 {
			return nil, fmt.Errorf("Ok(...) value must have a single type, got %d", len(ts))
		}
		failT := ast.NewBuiltinType(ast.TypeError)
		if tc.currentFunction != nil && len(tc.currentFunction.ReturnTypes) == 1 &&
			tc.currentFunction.ReturnTypes[0].IsResultType() && len(tc.currentFunction.ReturnTypes[0].TypeParams) >= 2 {
			failT = tc.currentFunction.ReturnTypes[0].TypeParams[1]
		}
		res := ast.NewResultType(ts[0], failT)
		tc.storeInferredType(e, []ast.TypeNode{res})
		return []ast.TypeNode{res}, nil

	case ast.ErrExprNode:
		ts, err := tc.inferExpressionType(e.Value)
		if err != nil {
			return nil, err
		}
		if len(ts) != 1 {
			return nil, fmt.Errorf("Err(...) value must have a single type, got %d", len(ts))
		}
		if tc.currentFunction == nil || len(tc.currentFunction.ReturnTypes) != 1 ||
			!tc.currentFunction.ReturnTypes[0].IsResultType() || len(tc.currentFunction.ReturnTypes[0].TypeParams) < 2 {
			return nil, fmt.Errorf("Err(...) is only valid inside a function that returns Result(Success, Failure)")
		}
		rt := tc.currentFunction.ReturnTypes[0]
		failExpected := rt.TypeParams[1]
		if !tc.IsTypeCompatible(ts[0], failExpected) {
			return nil, fmt.Errorf("Err(...) value type %s is not compatible with failure type %s",
				ts[0].String(), failExpected.String())
		}
		succ := rt.TypeParams[0]
		res := ast.NewResultType(succ, ts[0])
		tc.storeInferredType(e, []ast.TypeNode{res})
		return []ast.TypeNode{res}, nil

	default:
		tc.log.Tracef("Unhandled expression type: %T", expr)
		return nil, fmt.Errorf("cannot infer type for expression: %T", expr)
	}
}

// Checks if a method call is valid for a given type and returns its return type
func (tc *TypeChecker) inferMethodCallType(varType []ast.TypeNode, methodName string, args []ast.ExpressionNode) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"function":   "inferMethodCallType",
		"varType":    varType,
		"methodName": methodName,
		"args":       args,
	}).Tracef("inferMethodCallType")

	if len(varType) != 1 {
		tc.log.WithFields(logrus.Fields{
			"function": "inferMethodCallType",
			"varType":  varType,
		}).Tracef("Method calls are only valid on single types, got %s", formatTypeList(varType))
		return nil, fmt.Errorf("method calls are only valid on single types, got %s", formatTypeList(varType))
	}

	returnType, err := tc.CheckBuiltinMethod(varType[0], methodName, args)
	if err != nil {
		tc.log.WithFields(logrus.Fields{
			"function":   "inferMethodCallType",
			"varType":    varType,
			"methodName": methodName,
			"args":       args,
		}).Tracef("Error checking built-in method: %v", err)
		return nil, err
	}

	tc.log.WithFields(logrus.Fields{
		"function":   "inferMethodCallType",
		"varType":    varType,
		"methodName": methodName,
		"args":       args,
	}).Tracef("Successfully inferred method call type: %v", returnType)
	return returnType, nil
}
