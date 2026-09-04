package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	"go/types"

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

	if assign.CompoundOp != "" {
		if assign.IsShort {
			return fmt.Errorf("cannot use compound assignment with :=")
		}
		if len(assign.LValues) != 1 || len(assign.RValues) != 1 {
			return fmt.Errorf("compound assignment requires a single left- and right-hand side")
		}
		for _, et := range assign.ExplicitTypes {
			if et != nil {
				return fmt.Errorf("compound assignment does not support explicit types")
			}
		}
		return tc.inferCompoundAssignmentTypes(assign)
	}

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
			if !nValueGo && len(parts) == 1 {
				argTypes := make([][]ast.TypeNode, 0, len(fc.Arguments))
				for _, arg := range fc.Arguments {
					ts, err := tc.inferExpressionType(arg)
					if err != nil {
						return err
					}
					argTypes = append(argTypes, ts)
				}
				raw, found, err := tc.trySamePackageGoCall(string(fc.Function.ID), fc, argTypes, false)
				if err != nil {
					return err
				}
				if found && len(raw) == len(assign.LValues) {
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

	if !nValueGo {
		resolvedTypes = make([][]ast.TypeNode, 0, len(assign.RValues))
		for i, rvalue := range assign.RValues {
			var expected *ast.TypeNode
			if len(assign.ExplicitTypes) > i && assign.ExplicitTypes[i] != nil {
				expected = assign.ExplicitTypes[i]
			}
			var types []ast.TypeNode
			var err error
			if expected != nil {
				types, err = tc.inferExpressionTypeWithExpected(rvalue, expected)
			} else {
				types, err = tc.inferExpressionType(rvalue)
			}
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
				parts := strings.Split(string(l.Ident.ID), ".")
				if len(parts) > 1 {
					lhsType, _, _, err := tc.lookupVariableForExpression(&l, tc.CurrentScope())
					if err != nil {
						return err
					}
					if len(resolvedTypes[i]) != 1 {
						return fmt.Errorf("field assignment: right-hand side must have a single type")
					}
					lhsGo := tc.goTypeForExpression(l)
					if lhsGo == nil {
						lhsGo = tc.goTypeForForstType(lhsType)
					}
					var rhsGo types.Type
					if i < len(assign.RValues) {
						rhsGo = tc.goTypeForExpression(assign.RValues[i])
					}
					if rhsGo == nil {
						rhsGo = tc.goTypeForForstType(resolvedTypes[i][0])
					}
					if lhsGo != nil && rhsGo != nil && types.AssignableTo(rhsGo, lhsGo) {
						tc.storeInferredType(l, []ast.TypeNode{lhsType})
						break
					}
					if !tc.IsTypeCompatible(resolvedTypes[i][0], lhsType) {
						return fmt.Errorf("assignment type mismatch: cannot assign %s to %s (expected %s)",
							resolvedTypes[i][0].Ident, l.Ident.ID, lhsType.Ident)
					}
					tc.storeInferredType(l, []ast.TypeNode{lhsType})
					break
				}
				_, exists := tc.scopeStack.LookupVariableType(l.Ident.ID)
				if !exists {
					return fmt.Errorf("assignment to undeclared variable '%s' is not allowed; use 'var' or ':='", l.Ident.ID)
				}
				if tc.isPackageConst(l.Ident.ID) {
					return fmt.Errorf("cannot assign to const %s", l.Ident.ID)
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

			// Typed var/init: check RHS against explicit type (literal-union membership included).
			if isVarDeclaration && len(assign.RValues) > 0 && i < len(resolvedTypes) && len(resolvedTypes[i]) == 1 {
				rhs := resolvedTypes[i][0]
				lhs := l.ExplicitType
				ok := false
				if lit, isLit := expressionLiteralValue(assign.RValues[i]); isLit && tc.isLiteralUnionType(lhs) {
					ok = tc.literalAssignableToType(lit, lhs)
				} else {
					ok = tc.IsTypeCompatible(rhs, lhs)
				}
				if !ok {
					return fmt.Errorf("assignment type mismatch: cannot assign %s to %s (expected %s)",
						rhs.Ident, l.Ident.ID, lhs.Ident)
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

		case ast.DereferenceNode:
			if assign.IsShort {
				return fmt.Errorf("cannot use := on dereferenced assignment")
			}
			lhsTypes, err := tc.inferDerefExpressionAsAssignTarget(l)
			if err != nil {
				return err
			}
			if len(lhsTypes) != 1 || len(resolvedTypes[i]) != 1 {
				return fmt.Errorf("dereference assignment: expected single type on both sides")
			}
			if !tc.IsTypeCompatible(resolvedTypes[i][0], lhsTypes[0]) {
				return fmt.Errorf("assignment type mismatch: cannot assign %s through pointer (expected %s)",
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

	// Phase 4d: record aliases for short decls and reassignments.
	for i, lv := range assign.LValues {
		var rhs ast.ExpressionNode
		if i < len(assign.RValues) {
			rhs = assign.RValues[i]
		} else if len(assign.RValues) == 1 {
			rhs = assign.RValues[0]
		}
		if rhs == nil {
			continue
		}
		var rhsTypes []ast.TypeNode
		if i < len(resolvedTypes) {
			rhsTypes = resolvedTypes[i]
		}
		tc.recordAssignmentAlias(lv, rhs, rhsTypes)
		// Bind closure capture summary when assigning a function literal.
		if lit, ok := rhs.(ast.FunctionLiteralNode); ok {
			tc.bindClosureCaptures(lv, lit)
		}
	}

	// Phase 4b: writes are legal; drop overlapping refinement facts.
	// Short decls introduce bindings (alias recorded above); still invalidate when
	// rebinding an existing name (IsShort false) or writing through fields/indexes.
	if !assign.IsShort {
		for _, lv := range assign.LValues {
			writePath, span := tc.writePathFromAssignTarget(lv)
			if writePath != nil {
				tc.applyWriteInvalidation(writePath, span)
			}
		}
	} else {
		// := of a function literal: do not treat as a mutating write.
		for i := range assign.LValues {
			if i < len(assign.RValues) {
				if _, ok := assign.RValues[i].(ast.FunctionLiteralNode); ok {
					continue
				}
			}
		}
	}

	return nil
}

func (tc *TypeChecker) applyWriteInvalidation(writePath *AccessPath, span ast.SourceSpan) {
	if tc == nil || writePath == nil {
		return
	}
	if tc.capturingClosure {
		tc.pendingClosureWrites = append(tc.pendingClosureWrites, writePath)
		return
	}
	tc.invalidateOverlappingFacts(writePath, span)
	tc.recordBranchOrLoopWrite(writePath, span)
	tc.recordParamWriteDuringInfer(writePath)
}

func (tc *TypeChecker) writePathFromAssignTarget(lv ast.ExpressionNode) (*AccessPath, ast.SourceSpan) {
	switch l := lv.(type) {
	case ast.VariableNode:
		return tc.AccessPathForVariable(&l), l.Ident.Span
	case *ast.VariableNode:
		if l == nil {
			return nil, ast.SourceSpan{}
		}
		return tc.AccessPathForVariable(l), l.Ident.Span
	case ast.FieldAccessNode:
		id := dottedIdentFromExpr(l)
		vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(id), Span: l.Field.Span}}
		return tc.AccessPathForVariable(&vn), l.Field.Span
	case *ast.FieldAccessNode:
		if l == nil {
			return nil, ast.SourceSpan{}
		}
		id := dottedIdentFromExpr(*l)
		vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(id), Span: l.Field.Span}}
		return tc.AccessPathForVariable(&vn), l.Field.Span
	case ast.IndexExpressionNode:
		return tc.writePathFromIndexAssign(l)
	case *ast.IndexExpressionNode:
		if l == nil {
			return nil, ast.SourceSpan{}
		}
		return tc.writePathFromIndexAssign(*l)
	case ast.DereferenceNode:
		// Pointee write does not clobber Present on the pointer slot (RFC 13 §21).
		inner := ""
		switch v := l.Value.(type) {
		case ast.VariableNode:
			inner = string(v.Ident.ID)
		case *ast.VariableNode:
			if v != nil {
				inner = string(v.Ident.ID)
			}
		}
		if inner == "" {
			return nil, ast.SourceSpan{}
		}
		vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(inner)}}
		base := tc.AccessPathForVariable(&vn)
		if base == nil {
			return nil, ast.SourceSpan{}
		}
		deref := tc.paths.Intern(AccessPath{Root: base.Root, Steps: append(base.CloneSteps(), AccessStep{Kind: AccessDeref})})
		return deref, ast.SourceSpan{}
	default:
		return nil, ast.SourceSpan{}
	}
}

func (tc *TypeChecker) writePathFromIndexAssign(idx ast.IndexExpressionNode) (*AccessPath, ast.SourceSpan) {
	baseID := dottedIdentFromExpr(idx.Target)
	if baseID == "" {
		return nil, ast.SourceSpan{}
	}
	vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(baseID)}}
	base := tc.AccessPathForVariable(&vn)
	if base == nil {
		return nil, ast.SourceSpan{}
	}
	// users[0].age — IndexExpression may nest FieldAccess as target of outer assign
	// handled via VariableNode dotted ids; here target[index] → target[*].
	star := tc.paths.Intern(AccessPath{
		Root:  base.Root,
		Steps: append(base.CloneSteps(), AccessStep{Kind: AccessIndexAny}),
	})
	span := ast.SourceSpan{}
	if vn2, ok := idx.Target.(ast.VariableNode); ok {
		span = vn2.Ident.Span
	}
	return star, span
}
