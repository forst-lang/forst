package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/goload"
	"strconv"
	"strings"

	"go/types"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferExpressionType(expr ast.Node) ([]ast.TypeNode, error) {
	if tc.log.IsLevelEnabled(logrus.DebugLevel) {
		tc.log.WithFields(logrus.Fields{
			"function": "inferExpressionType",
			"expr":     expr,
		}).Debugf("Starting type inference for expression")
	}
	if isLiteralExpression(expr) {
		if cached, ok, err := tc.lookupCachedExpressionTypes(expr); err != nil {
			return nil, err
		} else if ok {
			tc.storeInferredType(expr, cached)
			return cached, nil
		}
	}
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:
		if e.Operator == ast.TokenArrow {
			// channel <- value
			if _, err := tc.inferExpressionType(e.Left); err != nil {
				return nil, err
			}
			if _, err := tc.inferExpressionType(e.Right); err != nil {
				return nil, err
			}
			span := ast.SourceSpan{}
			if vn, ok := e.Left.(ast.VariableNode); ok {
				span = vn.Ident.Span
			}
			tc.invalidateAfterChannelSend(e.Right, span)
			tc.storeInferredType(e, []ast.TypeNode{{Ident: ast.TypeVoid}})
			return []ast.TypeNode{{Ident: ast.TypeVoid}}, nil
		}
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

	case ast.RuneLiteralNode:
		typ := ast.TypeNode{Ident: ast.TypeInt}
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
				elSpan := spanOfExpression(el)
				return nil, reportf(elSpan, "array-element-type",
					fmt.Sprintf("array element %d must have a single type", i),
					fmt.Sprintf("Element %d of the array literal must infer to exactly one type.", i),
					"ensure each element has one type")
			}
			if i == 0 {
				elemType = ts[0]
			} else if elemType.Ident != ts[0].Ident {
				elSpan := spanOfExpression(el)
				return nil, reportf(elSpan, "array-mixed-types",
					"array literal has mixed element types",
					fmt.Sprintf("Array elements must share one type; got `%s` and `%s`.", formatTypeIdentForDiag(elemType.Ident), formatTypeIdentForDiag(ts[0].Ident)),
					"use the same type for every element or add an explicit element type")
			}
		}
		if e.Type.Ident != ast.TypeImplicit && e.Type.Ident != "" {
			elemType = e.Type
		}
		arr := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elemType}}
		tc.storeInferredType(e, []ast.TypeNode{arr})
		return []ast.TypeNode{arr}, nil

	case ast.VariableNode:
		parts := strings.Split(string(e.Ident.ID), ".")
		if len(parts) == 2 {
			if types, ok := tc.resolveForstSiblingQualifiedVar(parts[0], parts[1]); ok {
				tc.storeInferredType(e, types)
				return types, nil
			}
		}
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
			return nil, reportf(spanIndexExpr(e), "index-target-type",
				"index target must have a single type",
				fmt.Sprintf("The indexed value has %d types; subscript needs exactly one.", len(targetTypes)),
				"bind the container to a single-typed name")
		}
		t := targetTypes[0]
		indexTypes, err := tc.inferExpressionType(e.Index)
		if err != nil {
			return nil, err
		}
		if len(indexTypes) != 1 {
			return nil, reportf(spanIndexExpr(e), "index-type",
				"index expression must have a single index type",
				"The index expression must infer to exactly one type.",
				"use an Int index for slices/arrays/strings or the map key type")
		}
		if t.Ident == ast.TypeMap && len(t.TypeParams) >= 2 {
			wantK, wantV := t.TypeParams[0], t.TypeParams[1]
			if !tc.IsTypeCompatible(indexTypes[0], wantK) {
				return nil, reportf(spanIndexExpr(e), "index-type",
					"map index key type mismatch",
					fmt.Sprintf("Map key type must be `%s`, got `%s`.", formatTypeIdentForDiag(wantK.Ident), formatTypeIdentForDiag(indexTypes[0].Ident)),
					"convert the key or change the map's key type")
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
		if t.Ident == ast.TypeString {
			if indexTypes[0].Ident != ast.TypeInt {
				return nil, reportf(spanIndexExpr(e), "index-type",
					"string index must be Int",
					fmt.Sprintf("String indexing requires Int, got `%s`.", indexTypes[0].Ident),
					"use an integer index")
			}
			elem := ast.TypeNode{Ident: ast.TypeInt}
			tc.storeInferredType(e, []ast.TypeNode{elem})
			return []ast.TypeNode{elem}, nil
		}
		if t.Ident == ast.TypeBytes {
			if indexTypes[0].Ident != ast.TypeInt {
				return nil, reportf(spanIndexExpr(e), "index-type",
					"[]byte index must be Int",
					fmt.Sprintf("`[]byte` indexing requires Int, got `%s`.", indexTypes[0].Ident),
					"use an integer index")
			}
			elem := ast.TypeNode{Ident: ast.TypeIdent("byte")}
			tc.storeInferredType(e, []ast.TypeNode{elem})
			return []ast.TypeNode{elem}, nil
		}
		if t.Ident != ast.TypeArray || len(t.TypeParams) < 1 {
			return nil, reportf(spanIndexExpr(e), "index-target-type",
				"index target must be map, slice, or array",
				fmt.Sprintf("Cannot index type `%s`; expected map, slice, array, string, or []byte.", formatTypeIdentForDiag(t.Ident)),
				"index a supported container type")
		}
		if indexTypes[0].Ident != ast.TypeInt {
			return nil, reportf(spanIndexExpr(e), "index-type",
				"slice/array index must be Int",
				fmt.Sprintf("Slice and array indexes must be Int, got `%s`.", indexTypes[0].Ident),
				"use an integer index")
		}
		if err := checkFixedArrayIndexBounds(t, e.Index); err != nil {
			return nil, err
		}
		elem := t.TypeParams[0]
		tc.storeInferredType(e, []ast.TypeNode{elem})
		return []ast.TypeNode{elem}, nil

	case ast.SliceExpressionNode:
		if goT := tc.goTypeForExpression(e.Target); goT != nil {
			var elem types.Type
			switch u := goT.Underlying().(type) {
			case *types.Slice:
				elem = u.Elem()
			case *types.Array:
				elem = u.Elem()
			default:
				return nil, reportf(spanSliceExpr(e), "slice-target-type",
					"slice target must be a slice or array",
					fmt.Sprintf("Cannot slice type `%s`; expected a slice or array.", goT.String()),
					"slice a []T or [N]T value")
			}
			if e.Low != nil {
				lowTypes, err := tc.inferExpressionType(e.Low)
				if err != nil {
					return nil, err
				}
				if len(lowTypes) != 1 || lowTypes[0].Ident != ast.TypeInt {
					return nil, reportf(spanSliceExpr(e), "slice-bound",
						"slice low bound must be Int",
						"The low bound of a slice expression must be Int.",
						"use an integer low bound")
				}
			}
			if e.High != nil {
				highTypes, err := tc.inferExpressionType(e.High)
				if err != nil {
					return nil, err
				}
				if len(highTypes) != 1 || highTypes[0].Ident != ast.TypeInt {
					return nil, reportf(spanSliceExpr(e), "slice-bound",
						"slice high bound must be Int",
						"The high bound of a slice expression must be Int.",
						"use an integer high bound")
				}
			}
			ft, ok := tc.mapGoType(types.NewSlice(elem))
			if !ok {
				return nil, reportf(spanSliceExpr(e), "slice-target-type",
					"cannot map Go slice element type",
					"The Go slice element type could not be mapped to a Forst type.",
					"slice a supported Go slice or array type")
			}
			tc.storeInferredType(e, []ast.TypeNode{ft})
			return []ast.TypeNode{ft}, nil
		}
		targetTypes, err := tc.inferExpressionType(e.Target)
		if err != nil {
			return nil, err
		}
		if len(targetTypes) != 1 || targetTypes[0].Ident != ast.TypeArray || len(targetTypes[0].TypeParams) < 1 {
			return nil, reportf(spanSliceExpr(e), "slice-target-type",
				"slice target must be a slice or array",
				"The slice target must be a Forst slice or array type.",
				"slice a []T or [N]T value")
		}
		if e.Low != nil {
			lowTypes, err := tc.inferExpressionType(e.Low)
			if err != nil {
				return nil, err
			}
			if len(lowTypes) != 1 || lowTypes[0].Ident != ast.TypeInt {
				return nil, reportf(spanSliceExpr(e), "slice-bound",
					"slice low bound must be Int",
					"The low bound of a slice expression must be Int.",
					"use an integer low bound")
			}
		}
		if e.High != nil {
			highTypes, err := tc.inferExpressionType(e.High)
			if err != nil {
				return nil, err
			}
			if len(highTypes) != 1 || highTypes[0].Ident != ast.TypeInt {
				return nil, reportf(spanSliceExpr(e), "slice-bound",
					"slice high bound must be Int",
					"The high bound of a slice expression must be Int.",
					"use an integer high bound")
			}
		}
		elem := targetTypes[0].TypeParams[0]
		out := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elem}}
		tc.storeInferredType(e, []ast.TypeNode{out})
		return []ast.TypeNode{out}, nil

	case ast.SpreadExpressionNode:
		ts, err := tc.inferExpressionType(e.Expr)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, ts)
		return ts, nil

	case ast.FieldAccessNode:
		targetTypes, err := tc.inferExpressionType(e.Target)
		if err != nil {
			return nil, err
		}
		if len(targetTypes) == 1 && targetTypes[0].IsTupleType() {
			idx, convErr := strconv.Atoi(string(e.Field.ID))
			if convErr != nil || idx < 0 || idx >= len(targetTypes[0].TypeParams) {
				return nil, reportBodyf(e.Field.Span, "tuple-index", "tuple index %s out of range for %s", e.Field.ID, targetTypes[0].String())
			}
			ft := targetTypes[0].TypeParams[idx]
			tc.storeInferredType(e, []ast.TypeNode{ft})
			return []ast.TypeNode{ft}, nil
		}
		if goRecv := tc.goTypeForExpression(e.Target); goRecv != nil {
			obj, _, _ := types.LookupFieldOrMethod(goRecv, false, nil, string(e.Field.ID))
			if obj == nil {
				return nil, reportBodyf(e.Field.Span, "go-field", "%s has no field %s", goRecv.String(), e.Field.ID)
			}
			ft, ok := tc.mapGoType(obj.Type())
			if !ok {
				return nil, reportBodyf(e.Field.Span, "go-field", "cannot map Go field type %s", obj.Type().String())
			}
			tc.storeInferredType(e, []ast.TypeNode{ft})
			return []ast.TypeNode{ft}, nil
		}
		return nil, reportBodyf(e.Field.Span, "field-access", "field access on non-Go expression is not supported")

	case ast.MethodCallNode:
		argTypes := make([][]ast.TypeNode, 0, len(e.Arguments))
		for _, arg := range e.Arguments {
			ts, err := tc.inferExpressionType(arg)
			if err != nil {
				return nil, err
			}
			argTypes = append(argTypes, ts)
		}
		if goRecv := tc.goTypeForExpression(e.Receiver); goRecv != nil {
			fc := ast.FunctionCallNode{Arguments: e.Arguments, CallSpan: e.CallSpan, ArgSpans: e.ArgSpans}
			ret, err := tc.checkGoMethodCall(goRecv, e.Method, fc, argTypes, true)
			if err != nil {
				return nil, err
			}
			span := e.CallSpan
			if !span.IsSet() {
				span = e.Method.Span
			}
			tc.invalidateReachableMutableArg(e.Receiver, span, dropByForeign)
			tc.storeInferredType(e, ret)
			return ret, nil
		}
		if vn, ok := e.Receiver.(ast.VariableNode); ok {
			recvTypes, err := tc.inferExpressionType(e.Receiver)
			if err != nil {
				return nil, err
			}
			fc := ast.FunctionCallNode{Function: ast.Ident{ID: ast.Identifier(string(vn.Ident.ID) + "." + string(e.Method.ID))}, Arguments: e.Arguments, CallSpan: e.CallSpan, ArgSpans: e.ArgSpans}
			ret, err := tc.inferMethodCallType(vn.Ident.ID, recvTypes, string(e.Method.ID), fc, argTypes)
			if err != nil {
				return nil, err
			}
			tc.storeInferredType(e, ret)
			return ret, nil
		}
		sp := e.Method.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, reportBodyf(sp, "undefined-identifier", "method %s on receiver type %T", e.Method.ID, e.Receiver)

	case ast.FunctionCallNode:
		if e.Callee != nil {
			ret, err := tc.inferCalleeCall(e.Callee, e.Arguments, e.ArgSpans, e.CallSpan)
			if err != nil {
				return nil, err
			}
			tc.storeInferredType(e, ret)
			return ret, nil
		}

		tc.log.WithFields(logrus.Fields{
			"function": "inferExpressionType",
			"expr":     expr,
		}).Tracef("Checking function call: %s with %d arguments", e.Function.ID, len(e.Arguments))

		var argTypes [][]ast.TypeNode
		if signature, exists := tc.Functions[e.Function.ID]; exists {
			argTypes = make([][]ast.TypeNode, len(e.Arguments))
			for i, arg := range e.Arguments {
				exp := expectedTypeForCallParam(signature.Parameters, i)
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
			callSig := signature
			callSpan := e.CallSpan
			if !callSpan.IsSet() {
				callSpan = e.Function.Span
			}
			if len(e.TypeArgs) > 0 && len(signature.TypeParams) == 0 {
				_, err := tc.instantiateGenericCallExplicit(signature, e.TypeArgs, argTypes, callSpan)
				if err != nil {
					return nil, err
				}
			}
			if len(signature.TypeParams) > 0 {
				var inst FunctionSignature
				var err error
				if len(e.TypeArgs) > 0 {
					inst, err = tc.instantiateGenericCallExplicit(signature, e.TypeArgs, argTypes, callSpan)
				} else {
					inst, err = tc.instantiateGenericCall(signature, argTypes, callSpan)
				}
				if err != nil {
					return nil, err
				}
				callSig = inst
			}
			if err := tc.checkUserFunctionCall(e.Function.ID, callSig, e, argTypes); err != nil {
				return nil, err
			}
			tc.recordFunctionCall(e.Function.ID, callSpan)
			retTypes := callSig.ReturnTypes
			tc.storeInferredType(e, retTypes)
			return retTypes, nil
		}

		// Resolution order: Forst func → same-package Go → type guard → local → qualified → dot-import → builtin.
		if !strings.Contains(string(e.Function.ID), ".") {
			if ret, found, err := tc.trySamePackageGoCall(string(e.Function.ID), e, argTypes, true); err != nil {
				return nil, err
			} else if found {
				tc.invalidateAfterUntrustedGoCall(e)
				tc.storeInferredType(e, ret)
				return ret, nil
			}
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

		// First check if this is a local variable holding a function value.
		if varType, exists := tc.scopeStack.LookupVariableType(e.Function.ID); exists {
			if len(varType) == 1 && varType[0].IsFunctionType() {
				ret, err := tc.inferCalleeCall(ast.VariableNode{Ident: e.Function}, e.Arguments, e.ArgSpans, e.CallSpan)
				if err != nil {
					return nil, err
				}
				tc.storeInferredType(e, ret)
				return ret, nil
			}
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

			// Forst sibling package in same module (before Go stub resolution)
			if ret, err := tc.resolveForstSiblingCall(pkgName, funcName, e, argTypes); err != nil {
				return nil, err
			} else if ret != nil {
				tc.storeInferredType(e, ret)
				return ret, nil
			}

			// Node TypeScript import (before Go package resolution)
			if ret, found, err := tc.tryNodeQualifiedCall(pkgName, funcName, e, argTypes); err != nil {
				return nil, err
			} else if found {
				tc.storeInferredType(e, ret)
				return ret, nil
			}

			// Imported Go package (Forst↔Go boundary): batch or lazy go/packages load
			if gp := tc.goPackageForImportLocal(pkgName); gp != nil {
				ret, err := tc.checkGoQualifiedCall(gp, pkgName, funcName, e, argTypes, true)
				if err != nil {
					return nil, err
				}
				callSpan := e.CallSpan
				if !callSpan.IsSet() {
					callSpan = e.Function.Span
				}
				tc.recordCrossPackageCall(pkgName, ast.Identifier(funcName), callSpan)
				tc.invalidateAfterUntrustedGoCall(e)
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
			if tc.IsImportedLocalName(pkgName) && !tc.isNodeImportLocal(pkgName) {
				callSpan := e.CallSpan
				if !callSpan.IsSet() {
					callSpan = e.Function.Span
				}
				importPath := pkgName
				if tc.importPathByLocal != nil {
					if p, ok := tc.importPathByLocal[pkgName]; ok && p != "" {
						importPath = p
					}
				}
				if loadErr := tc.goImportLoadErrorForPath(importPath); loadErr != nil {
					return nil, reportBodyf(callSpan, "go-import", "failed to load Go package %q: %v", importPath, loadErr)
				}
				tc.log.WithFields(logrus.Fields{
					"function":           "inferExpressionType",
					"pkgLocal":           pkgName,
					"importPath":         importPath,
					"goWorkspaceDir":     tc.GoWorkspaceDir,
					"goPackagesPreloaded": tc.goPackagesPreloaded,
					"missingGoImports":   tc.missingGoImportPaths(),
				}).Debug("Go import package types not loaded")
				return nil, reportBodyf(callSpan, "go-import", "%s", goload.GoImportTypesNotLoadedMsg(pkgName, importPath, tc.GoWorkspaceDir, tc.NodeBoundaryRoot))
			}
		} else if len(parts) >= 3 {
			base := ast.Identifier(parts[0])
			if gt, ok := tc.variableGoTypes[base]; ok && gt != nil {
				fieldPath := parts[1 : len(parts)-1]
				recvGo, err := goTypeAtFieldPath(gt, fieldPath)
				if err == nil {
					fc := ast.FunctionCallNode{
						Function:  e.Function,
						Arguments: e.Arguments,
						CallSpan:  e.CallSpan,
						ArgSpans:  e.ArgSpans,
					}
					ret, err := tc.checkGoMethodCall(recvGo, e.Function, fc, argTypes, true)
					if err != nil {
						return nil, err
					}
					tc.storeInferredType(e, ret)
					return ret, nil
				}
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

		if ret, ok, err := tc.inferNominalErrorConstructorCall(e, argTypes); ok {
			if err != nil {
				return nil, err
			}
			tc.storeInferredType(e, ret)
			return ret, nil
		}

		// Phase 4f: unresolved pkg.Func — invalidate, do not reject.
		if ret, ok := tc.treatUnresolvedQualifiedCallAsForeign(e); ok {
			tc.storeInferredType(e, ret)
			return ret, nil
		}

		sp := e.Function.Span
		if !sp.IsSet() {
			sp = e.CallSpan
		}
		return nil, reportBodyf(sp, "undefined-identifier", "unknown identifier: %s", e.Function.ID)

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
			return nil, reportf(spanOfExpression(e.Value), "deref-type",
				"dereference requires a single pointer type",
				fmt.Sprintf("Dereference is only valid on a single type, got %s.", formatTypeList(valueType)),
				"use `*p` where `p` has exactly one pointer type")
		}
		tc.log.Tracef("Dereference type identifier: %+v", valueType[0].Node)
		if valueType[0].Ident != ast.TypePointer {
			return nil, reportf(spanOfExpression(e.Value), "deref-type",
				"dereference requires a pointer type",
				fmt.Sprintf("Cannot dereference type `%s`; expected a pointer.", valueType[0].Ident),
				"use `*p` on a pointer value")
		}
		tc.log.Tracef("Dereference type: %+v", valueType[0].TypeParams)
		tc.storeInferredType(e, valueType[0].TypeParams)
		return valueType[0].TypeParams, nil

	case ast.NilLiteralNode:
		// Return a special marker (empty slice) to indicate untyped nil; context must resolve
		return nil, nil

	case ast.FunctionLiteralNode:
		ret, err := tc.inferFunctionLiteral(e, expr)
		if err != nil {
			return nil, err
		}
		tc.storeInferredType(e, ret)
		return ret, nil

	case ast.IotaLiteralNode:
		return nil, reportf(spanOfNode(e), "iota-outside-const",
			"iota is only valid in const declarations",
			"The predeclared `iota` may only appear in a const group.",
			"move `iota` into a const declaration")

	case ast.TypeExpressionNode:
		tc.storeInferredType(e, []ast.TypeNode{e.Type})
		return []ast.TypeNode{e.Type}, nil

	case ast.MapLiteralNode:
		mapSpan := ast.SourceSpan{}
		if len(e.Entries) > 0 {
			mapSpan = spanOfExpression(e.Entries[0].Key)
		}
		if e.Type.Ident != ast.TypeMap || len(e.Type.TypeParams) != 2 {
			return nil, reportf(mapSpan, "map-literal-type",
				"map literal has invalid type",
				fmt.Sprintf("Map literal type must be Map(K, V); got %v.", e.Type),
				"annotate the literal as Map(keyType, valueType)")
		}
		wantK, wantV := e.Type.TypeParams[0], e.Type.TypeParams[1]
		for i, ent := range e.Entries {
			entrySpan := firstSetSpan(spanOfExpression(ent.Key), spanOfExpression(ent.Value), mapSpan)
			kt, err := tc.inferExpressionType(ent.Key)
			if err != nil {
				return nil, err
			}
			if len(kt) != 1 {
				return nil, reportf(entrySpan, "map-literal-key-type",
					fmt.Sprintf("map literal entry %d key must have one type", i),
					fmt.Sprintf("Key %d must infer to exactly one type.", i),
					"ensure each map key has a single type")
			}
			if !tc.IsTypeCompatible(kt[0], wantK) {
				return nil, reportf(entrySpan, "map-literal-key-type",
					fmt.Sprintf("map literal entry %d key type mismatch", i),
					fmt.Sprintf("Key %d must have type `%s`, got `%s`.", i, formatTypeIdentForDiag(wantK.Ident), formatTypeIdentForDiag(kt[0].Ident)),
					"convert the key or change the map key type")
			}
			vt, err := tc.inferExpressionType(ent.Value)
			if err != nil {
				return nil, err
			}
			if len(vt) != 1 {
				return nil, reportf(entrySpan, "map-literal-value-type",
					fmt.Sprintf("map literal entry %d value must have one type", i),
					fmt.Sprintf("Value %d must infer to exactly one type.", i),
					"ensure each map value has a single type")
			}
			if !tc.IsTypeCompatible(vt[0], wantV) {
				return nil, reportf(entrySpan, "map-literal-value-type",
					fmt.Sprintf("map literal entry %d value type mismatch", i),
					fmt.Sprintf("Value %d must have type `%s`, got `%s`.", i, formatTypeIdentForDiag(wantV.Ident), formatTypeIdentForDiag(vt[0].Ident)),
					"convert the value or change the map value type")
			}
		}
		tc.storeInferredType(e, []ast.TypeNode{e.Type})
		return []ast.TypeNode{e.Type}, nil

	case ast.OkExprNode:
		sp := e.Span
		if !sp.IsSet() {
			sp = spanOfExpression(e.Value)
		}
		return nil, reportf(sp, "result-value-constructor",
			"Ok(...) is not a value constructor",
			"`Ok(...)` is a Result discriminant for `is` / `ensure`, not a runtime value constructor.",
			"use `if r is Ok()` / `ensure r is Ok()`, or return a plain success value of type S for Result(S, F)")

	case ast.ErrExprNode:
		sp := e.Span
		if !sp.IsSet() {
			sp = spanOfExpression(e.Value)
		}
		return nil, reportf(sp, "result-value-constructor",
			"Err(...) is not a value constructor",
			"`Err(...)` is a Result discriminant for `is` / `ensure`, not a runtime value constructor.",
			"use `is Err()` / `ensure ... is Err()` and FFI/interop for failure values")

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
		callSpan := firstSetSpan(e.CallSpan, e.Function.Span)
		tc.log.WithFields(logrus.Fields{
			"function": "inferMethodCallType",
			"varType":  varType,
		}).Tracef("Method calls are only valid on single types, got %s", formatTypeList(varType))
		return nil, reportf(callSpan, "method-receiver-type",
			"method call requires a single receiver type",
			fmt.Sprintf("Method calls require exactly one receiver type, got %s.", formatTypeList(varType)),
			"call the method on a single-typed receiver")
	}

	t := varType[0]
	callSpan := firstSetSpan(e.CallSpan, e.Function.Span)
	if t.IsResultType() && len(t.TypeParams) >= 2 {
		switch methodName {
		case "Ok":
			return []ast.TypeNode{t.TypeParams[0]}, nil
		case "Err":
			return []ast.TypeNode{t.TypeParams[1]}, nil
		default:
			return nil, reportf(callSpan, "method-undefined",
				fmt.Sprintf("method `%s()` not valid on Result", methodName),
				fmt.Sprintf("Result types only support `.Ok()` and `.Err()`; `%s()` is not valid.", methodName),
				"use `.Ok()` or `.Err()`, or call a method on the success/failure payload after narrowing")
		}
	}

	if goRecv, ok := tc.variableGoTypes[receiver]; ok && goRecv != nil {
		method := e.Function
		if method.ID == "" {
			method = ast.Ident{ID: ast.Identifier(methodName)}
		}
		ret, err := tc.checkGoMethodCall(goRecv, method, e, argTypes, true)
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

	if ret, err := tc.checkUserTypeMethod(t, methodName, e.Arguments, callSpan); err == nil {
		return ret, nil
	} else if tc.TypeMethods != nil {
		// Only fall through when the type has no method table; otherwise surface the error.
		if methods, ok := tc.TypeMethods[t.Ident]; ok && len(methods) > 0 {
			return nil, err
		}
	}

	if ret, err := tc.checkContractShapeMethod(t, methodName, e.Arguments, callSpan); err == nil {
		return ret, nil
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
		return nil, reportf(spanIndexExpr(e), "index-target-type",
			"index target must have a single type",
			fmt.Sprintf("The indexed value has %d types; subscript needs exactly one.", len(targetTypes)),
			"bind the container to a single-typed name")
	}
	t := targetTypes[0]
	indexTypes, err := tc.inferExpressionType(e.Index)
	if err != nil {
		return nil, err
	}
	if len(indexTypes) != 1 {
		return nil, reportf(spanIndexExpr(e), "index-type",
			"index expression must have a single index type",
			"The index expression must infer to exactly one type.",
			"use an Int index for slices/arrays/strings or the map key type")
	}
	if t.Ident == ast.TypeMap && len(t.TypeParams) >= 2 {
		wantK, wantV := t.TypeParams[0], t.TypeParams[1]
		if !tc.IsTypeCompatible(indexTypes[0], wantK) {
			return nil, reportf(spanIndexExpr(e), "index-type",
				"map index key type mismatch",
				fmt.Sprintf("Map key type must be `%s`, got `%s`.", formatTypeIdentForDiag(wantK.Ident), formatTypeIdentForDiag(indexTypes[0].Ident)),
				"convert the key or change the map's key type")
		}
		tc.storeInferredType(e, []ast.TypeNode{wantV})
		return []ast.TypeNode{wantV}, nil
	}
	if t.Ident == ast.TypeString {
		if indexTypes[0].Ident != ast.TypeInt {
			return nil, reportf(spanIndexExpr(e), "index-type",
				"string index must be Int",
				fmt.Sprintf("String indexing requires Int, got `%s`.", indexTypes[0].Ident),
				"use an integer index")
		}
		elem := ast.TypeNode{Ident: ast.TypeInt}
		tc.storeInferredType(e, []ast.TypeNode{elem})
		return []ast.TypeNode{elem}, nil
	}
	if t.Ident == ast.TypeBytes {
		if indexTypes[0].Ident != ast.TypeInt {
			return nil, reportf(spanIndexExpr(e), "index-type",
				"[]byte index must be Int",
				fmt.Sprintf("`[]byte` indexing requires Int, got `%s`.", indexTypes[0].Ident),
				"use an integer index")
		}
		elem := ast.TypeNode{Ident: ast.TypeIdent("byte")}
		tc.storeInferredType(e, []ast.TypeNode{elem})
		return []ast.TypeNode{elem}, nil
	}
	if t.Ident != ast.TypeArray || len(t.TypeParams) < 1 {
		return nil, reportf(spanIndexExpr(e), "index-target-type",
			"index target must be map, slice, or array",
			fmt.Sprintf("Cannot index type `%s`; expected map, slice, array, string, or []byte.", formatTypeIdentForDiag(t.Ident)),
			"index a supported container type")
	}
	if indexTypes[0].Ident != ast.TypeInt {
		return nil, reportf(spanIndexExpr(e), "index-type",
			"slice/array index must be Int",
			fmt.Sprintf("Slice and array indexes must be Int, got `%s`.", indexTypes[0].Ident),
			"use an integer index")
	}
	elem := t.TypeParams[0]
	tc.storeInferredType(e, []ast.TypeNode{elem})
	return []ast.TypeNode{elem}, nil
}

// inferDerefExpressionAsAssignTarget types *p = x (including **pp = x).
func (tc *TypeChecker) inferDerefExpressionAsAssignTarget(e ast.DereferenceNode) ([]ast.TypeNode, error) {
	ptrTypes, err := tc.inferExpressionType(e.Value)
	if err != nil {
		return nil, err
	}
	if len(ptrTypes) != 1 || ptrTypes[0].Ident != ast.TypePointer || len(ptrTypes[0].TypeParams) != 1 {
		return nil, reportf(spanOfExpression(e.Value), "deref-assign-type",
			"dereference assignment requires a pointer on the left",
			fmt.Sprintf("Left-hand side of `*p = v` must be a pointer, got %s.", formatTypeList(ptrTypes)),
			"dereference a pointer variable or field")
	}
	elem := ptrTypes[0].TypeParams[0]
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
		case ast.ArrayLiteralNode:
			if len(x.Value) == 0 && expected.Ident == ast.TypeArray && len(expected.TypeParams) == 1 {
				arr := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{expected.TypeParams[0]}}
				if expected.ArrayLen != nil {
					arr.ArrayLen = expected.ArrayLen
				}
				tc.storeInferredType(expr, []ast.TypeNode{arr})
				return []ast.TypeNode{arr}, nil
			}
			if expected.IsFixedArray() && len(expected.TypeParams) == 1 {
				arrSpan := ast.SourceSpan{}
				if len(x.Value) > 0 {
					arrSpan = spanOfExpression(x.Value[0])
				}
				if err := checkFixedArrayLiteralLength(*expected, len(x.Value), arrSpan); err != nil {
					return nil, err
				}
			}
			return tc.inferExpressionType(x)
		case ast.IndexExpressionNode:
			return tc.inferExpressionType(x)
		}
	}
	return tc.inferExpressionType(expr)
}
