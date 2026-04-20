package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferAssignmentTypes(assign ast.AssignmentNode) error {
	var resolvedTypes [][]ast.TypeNode

	tc.log.WithFields(logrus.Fields{
		"assignment": assign.String(),
		"lvalues":    len(assign.LValues),
		"rvalues":    len(assign.RValues),
		"function":   "inferAssignmentTypes",
	}).Trace("Starting type inference for assignment")

	nValueGo := false
	if len(assign.RValues) == 1 && len(assign.LValues) >= 2 {
		if fc, ok := assign.RValues[0].(ast.FunctionCallNode); ok {
			parts := strings.Split(string(fc.Function.ID), ".")
			if len(parts) == 2 {
				if gp := tc.goPackageForImportLocal(parts[0]); gp != nil {
					argTypes := make([][]ast.TypeNode, 0, len(fc.Arguments))
					for _, arg := range fc.Arguments {
						ts, err := tc.inferExpressionType(arg)
						if err != nil {
							return err
						}
						argTypes = append(argTypes, ts)
					}
					raw, err := tc.checkGoQualifiedCall(gp, parts[0], parts[1], fc, argTypes, false)
					if err == nil && len(raw) == len(assign.LValues) {
						resolvedTypes = make([][]ast.TypeNode, len(raw))
						for i := range raw {
							resolvedTypes[i] = []ast.TypeNode{raw[i]}
						}
						nValueGo = true
						tc.storeInferredType(fc, raw)
					}
				}
			}
			if !nValueGo && len(parts) == 1 && len(tc.dotImportPkgs) > 0 {
				sp := fc.Function.Span
				if !sp.IsSet() {
					sp = fc.CallSpan
				}
				gp, err := tc.lookupDotImportFunc(parts[0], sp)
				if err != nil {
					return err
				}
				if gp != nil {
					argTypes := make([][]ast.TypeNode, 0, len(fc.Arguments))
					for _, arg := range fc.Arguments {
						ts, err := tc.inferExpressionType(arg)
						if err != nil {
							return err
						}
						argTypes = append(argTypes, ts)
					}
					raw, err := tc.checkGoQualifiedCall(gp, gp.Path(), parts[0], fc, argTypes, false)
					if err == nil && len(raw) == len(assign.LValues) {
						resolvedTypes = make([][]ast.TypeNode, len(raw))
						for i := range raw {
							resolvedTypes[i] = []ast.TypeNode{raw[i]}
						}
						nValueGo = true
						tc.storeInferredType(fc, raw)
					}
				}
			}
		}
	}

	if !nValueGo {
		resolvedTypes = make([][]ast.TypeNode, 0, len(assign.RValues))
		for _, rvalue := range assign.RValues {
			types, err := tc.inferExpressionType(rvalue)
			if err != nil {
				return err
			}
			resolvedTypes = append(resolvedTypes, types)
		}
	}

	if tc.log != nil {
		tc.log.WithFields(logrus.Fields{
			"resolvedTypes": resolvedTypes,
			"LValues":       assign.LValues,
			"RValues":       assign.RValues,
		}).Debug("Before distributing function return types to LValues")
	}

	if len(assign.RValues) == 1 && len(resolvedTypes) == 1 && len(assign.LValues) > 1 {
		if _, ok := assign.RValues[0].(ast.FunctionCallNode); ok {
			if len(resolvedTypes[0]) == len(assign.LValues) {
				newResolved := make([][]ast.TypeNode, len(assign.LValues))
				for i := range assign.LValues {
					newResolved[i] = []ast.TypeNode{resolvedTypes[0][i]}
				}
				resolvedTypes = newResolved
			} else if len(resolvedTypes[0]) == 1 && len(assign.LValues) == 2 &&
				resolvedTypes[0][0].IsResultType() && len(resolvedTypes[0][0].TypeParams) >= 2 {
				// Result(S, F) is one Forst return type but maps to two LHS slots (success, failure).
				rt := resolvedTypes[0][0]
				resolvedTypes = [][]ast.TypeNode{
					{rt.TypeParams[0]},
					{rt.TypeParams[1]},
				}
			}
		}
	}

	// Forst intentionally does not support Go's map comma-ok form (v, ok := m[k]).
	if len(assign.RValues) == 1 && len(assign.LValues) == 2 {
		if idx, ok := assign.RValues[0].(ast.IndexExpressionNode); ok {
			targetTypes, err := tc.inferExpressionType(idx.Target)
			if err != nil {
				return err
			}
			if len(targetTypes) == 1 && targetTypes[0].Ident == ast.TypeMap {
				return fmt.Errorf("map index comma-ok assignment (v, ok := m[k]) is not supported")
			}
		}
	}

	if len(assign.RValues) > 0 && len(resolvedTypes) != len(assign.LValues) {
		return fmt.Errorf("assignment: %d left-hand values but right-hand produces %d value(s)", len(assign.LValues), len(resolvedTypes))
	}

	if tc.log != nil {
		tc.log.WithFields(logrus.Fields{
			"resolvedTypes": resolvedTypes,
		}).Debug("After distributing function return types to LValues")
	}

	for i, lv := range assign.LValues {
		switch l := lv.(type) {
		case ast.VariableNode:
			isVarDeclaration := len(assign.ExplicitTypes) > i && assign.ExplicitTypes[i] != nil

			if !assign.IsShort && !isVarDeclaration {
				_, exists := tc.scopeStack.LookupVariableType(l.Ident.ID)
				if !exists {
					return fmt.Errorf("assignment to undeclared variable '%s' is not allowed; use 'var' or ':='", l.Ident.ID)
				}
			}

			if isVarDeclaration {
				explicitType := l.ExplicitType
				isPointer := explicitType.Ident == ast.TypePointer
				isBuiltin := explicitType.Ident == ast.TypeString || explicitType.Ident == ast.TypeInt || explicitType.Ident == ast.TypeFloat || explicitType.Ident == ast.TypeBool || explicitType.Ident == ast.TypeError || explicitType.Ident == ast.TypeVoid || explicitType.Ident == ast.TypeArray || explicitType.Ident == ast.TypeMap || explicitType.Ident == ast.TypeShape || explicitType.Ident == ast.TypeObject
				_, isDefined := tc.Defs[explicitType.Ident]
				if !isPointer && !isBuiltin && !isDefined {
					return fmt.Errorf("undefined type name '%s' in variable declaration", explicitType.Ident)
				}
			}

			if isVarDeclaration && len(assign.RValues) == 1 {
				if _, isNil := assign.RValues[0].(ast.NilLiteralNode); isNil {
					explicitType := l.ExplicitType
					isPointer := explicitType.Ident == ast.TypePointer
					isInterface := explicitType.Ident == ast.TypeObject
					isMap := explicitType.Ident == ast.TypeMap
					isArray := explicitType.Ident == ast.TypeArray
					isFunc := explicitType.Ident == ast.TypeIdent("Func")
					if !isPointer && !isInterface && !isMap && !isArray && !isFunc {
						return fmt.Errorf("cannot assign nil to variable of type '%s'", explicitType.Ident)
					}
				}
			}

			if l.ExplicitType.Ident != "" && l.ExplicitType.Ident != ast.TypeImplicit {
				tc.log.WithFields(logrus.Fields{
					"variable":     l.Ident.ID,
					"explicitType": l.ExplicitType.Ident,
					"function":     "inferAssignmentTypes",
				}).Trace("Using explicit type for variable (alias preserved)")
				tc.storeInferredVariableType(l, []ast.TypeNode{l.ExplicitType})
				tc.storeInferredType(l, []ast.TypeNode{l.ExplicitType})
			} else {
				tc.log.WithFields(logrus.Fields{
					"variable":     l.Ident.ID,
					"resolvedType": resolvedTypes[i],
					"function":     "inferAssignmentTypes",
				}).Trace("Using resolved type for variable")
				tc.storeInferredVariableType(l, resolvedTypes[i])
				tc.storeInferredType(l, resolvedTypes[i])
			}

		case ast.IndexExpressionNode:
			if assign.IsShort {
				return fmt.Errorf("cannot use := on indexed assignment")
			}
			if len(assign.ExplicitTypes) > i && assign.ExplicitTypes[i] != nil {
				return fmt.Errorf("indexed assignment does not support explicit types on the left-hand side")
			}
			lhsTypes, err := tc.inferIndexExpressionAsAssignTarget(l)
			if err != nil {
				return err
			}
			if len(lhsTypes) != 1 {
				return fmt.Errorf("indexed assignment: left-hand side must have a single type")
			}
			if len(resolvedTypes[i]) != 1 {
				return fmt.Errorf("indexed assignment: right-hand side must have a single type")
			}
			if !tc.IsTypeCompatible(resolvedTypes[i][0], lhsTypes[0]) {
				return fmt.Errorf("assignment type mismatch: cannot assign %s to element (expected %s)",
					resolvedTypes[i][0].Ident, lhsTypes[0].Ident)
			}
			tc.storeInferredType(l, lhsTypes)

		default:
			return fmt.Errorf("unsupported assignment target type: %T", lv)
		}
	}

	tc.log.WithFields(logrus.Fields{
		"assignment":    assign.String(),
		"lvalues":       assign.LValues,
		"resolvedTypes": resolvedTypes,
		"function":      "inferAssignmentTypes",
	}).Trace("Finished type inference for assignment")

	tc.bindVariableGoTypesFromCall(assign)

	return nil
}
