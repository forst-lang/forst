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
		asSpan := spanOfNode(assign)
		if assign.IsShort {
			return reportf(asSpan, "compound-assign-short",
				"compound assignment cannot use `:=",
				"Compound operators like `+=` require an existing variable (`=`).",
				"declare the variable first, then use `=` with `+=` / `-=` / …")
		}
		if len(assign.LValues) != 1 || len(assign.RValues) != 1 {
			return reportf(asSpan, "compound-assign-arity",
				"compound assignment needs one value on each side",
				"Compound assignment requires exactly one left- and right-hand side.",
				"split into separate statements or use a single target")
		}
		for _, et := range assign.ExplicitTypes {
			if et != nil {
				return reportf(asSpan, "compound-assign-explicit-type",
					"compound assignment does not support explicit types",
					"Left-hand explicit types are not allowed with `+=`, `-=`, etc.",
					"drop the type annotation or use plain `=` assignment")
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
				return reportf(spanIndexExpr(idx), "map-comma-ok-unsupported",
					"map comma-ok assignment is not supported",
					"Forst does not support Go's `(v, ok := m[k])` map comma-ok form.",
					"use a single-value map read and handle missing keys with Result or explicit checks")
			}
		}
	}

	if len(assign.RValues) > 0 && len(resolvedTypes) != len(assign.LValues) {
		return reportf(spanOfNode(assign), "assignment-arity",
			"assignment arity mismatch",
			fmt.Sprintf("%d left-hand value(s) but the right-hand side produces %d value(s).", len(assign.LValues), len(resolvedTypes)),
			"match the number of names to the return values or split the assignment")
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
						return reportf(l.Ident.Span, "assignment-type",
							"field assignment needs a single right-hand type",
							"The right-hand side must infer to exactly one type for field assignment.",
							"return or bind a single value")
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
						return reportf(l.Ident.Span, "assignment-type-mismatch",
							"assignment type mismatch",
							fmt.Sprintf("Cannot assign `%s` to `%s` (expected `%s`).", resolvedTypes[i][0].Ident, l.Ident.ID, lhsType.Ident),
							"convert the value or change the field type")
					}
					tc.storeInferredType(l, []ast.TypeNode{lhsType})
					break
				}
				_, exists := tc.scopeStack.LookupVariableType(l.Ident.ID)
				if !exists {
					return reportf(l.Ident.Span, "assignment-undeclared",
						fmt.Sprintf("assignment to undeclared variable `%s`", l.Ident.ID),
						fmt.Sprintf("Variable `%s` is not declared; plain `=` requires an existing binding.", l.Ident.ID),
						"declare it with `var` or use `:=` for a new binding")
				}
				if tc.isPackageConst(l.Ident.ID) {
					return reportf(l.Ident.Span, "assign-to-const",
						fmt.Sprintf("cannot assign to const `%s`", l.Ident.ID),
						fmt.Sprintf("`%s` is a const and cannot be reassigned.", l.Ident.ID),
						"use a `var` binding instead")
				}
			}

			if isVarDeclaration {
				explicitType := l.ExplicitType
				isPointer := explicitType.Ident == ast.TypePointer
				isBuiltin := explicitType.Ident == ast.TypeString || explicitType.Ident == ast.TypeInt || explicitType.Ident == ast.TypeFloat || explicitType.Ident == ast.TypeBool || explicitType.Ident == ast.TypeError || explicitType.Ident == ast.TypeVoid || explicitType.Ident == ast.TypeArray || explicitType.Ident == ast.TypeMap || explicitType.Ident == ast.TypeShape || explicitType.Ident == ast.TypeObject
				_, isDefined := tc.Defs[explicitType.Ident]
				if !isPointer && !isBuiltin && !isDefined {
					return reportf(l.Ident.Span, "undefined-type",
						fmt.Sprintf("undefined type `%s`", explicitType.Ident),
						fmt.Sprintf("Type name `%s` in the variable declaration is not defined.", explicitType.Ident),
						"declare the type or use a built-in name")
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
						return reportf(l.Ident.Span, "nil-assign-type",
							fmt.Sprintf("cannot assign nil to `%s`", explicitType.Ident),
							fmt.Sprintf("Type `%s` cannot be initialized with `nil`.", explicitType.Ident),
							"use a pointer, map, slice, interface, or func type — or supply a value")
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
					return reportf(l.Ident.Span, "assignment-type-mismatch",
						"assignment type mismatch",
						fmt.Sprintf("Cannot assign `%s` to `%s` (expected `%s`).", rhs.Ident, l.Ident.ID, lhs.Ident),
						"convert the initializer or change the declared type")
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
			idxSpan := spanIndexExpr(l)
			if assign.IsShort {
				return reportf(idxSpan, "indexed-assign-short",
					"indexed assignment cannot use `:=",
					"Subscript assignment requires an existing container (`=`).",
					"declare the container first, then assign with `xs[i] = v`")
			}
			if len(assign.ExplicitTypes) > i && assign.ExplicitTypes[i] != nil {
				return reportf(idxSpan, "indexed-assign-explicit-type",
					"indexed assignment does not support explicit types",
					"Left-hand explicit types are not allowed on subscript targets.",
					"drop the type annotation on the index target")
			}
			lhsTypes, err := tc.inferIndexExpressionAsAssignTarget(l)
			if err != nil {
				return err
			}
			if len(lhsTypes) != 1 {
				return reportf(idxSpan, "indexed-assign-type",
					"indexed assignment left-hand side must have a single type",
					"The subscript target must refer to one element type.",
					"index a slice, array, map, or string element")
			}
			if len(resolvedTypes[i]) != 1 {
				return reportf(idxSpan, "indexed-assign-type",
					"indexed assignment right-hand side must have a single type",
					"The assigned value must have exactly one type.",
					"bind a single value on the right-hand side")
			}
			if !tc.IsTypeCompatible(resolvedTypes[i][0], lhsTypes[0]) {
				return reportf(idxSpan, "assignment-type-mismatch",
					"assignment type mismatch",
					fmt.Sprintf("Cannot assign `%s` to element type `%s`.", resolvedTypes[i][0].Ident, lhsTypes[0].Ident),
					"convert the value or change the container element type")
			}
			tc.storeInferredType(l, lhsTypes)

		case ast.DereferenceNode:
			derefSpan := spanOfExpression(l.Value)
			if assign.IsShort {
				return reportf(derefSpan, "deref-assign-short",
					"dereferenced assignment cannot use `:=",
					"Assignment through `*` requires an existing pointer variable (`=`).",
					"declare the pointer first, then use `*p = v`")
			}
			lhsTypes, err := tc.inferDerefExpressionAsAssignTarget(l)
			if err != nil {
				return err
			}
			if len(lhsTypes) != 1 || len(resolvedTypes[i]) != 1 {
				return reportf(derefSpan, "deref-assign-type",
					"dereference assignment expects single types on both sides",
					"Both the pointer target and the assigned value must have exactly one type.",
					"use `*p = v` where `p` is a pointer and `v` matches the pointee type")
			}
			if !tc.IsTypeCompatible(resolvedTypes[i][0], lhsTypes[0]) {
				return reportf(derefSpan, "assignment-type-mismatch",
					"assignment type mismatch",
					fmt.Sprintf("Cannot assign `%s` through pointer (expected `%s`).", resolvedTypes[i][0].Ident, lhsTypes[0].Ident),
					"convert the value or change the pointee type")
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
