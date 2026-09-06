package typechecker

import (
	"fmt"
	"strings"

	"forst/internal/ast"

	"go/types"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferAssignmentLValues(assign ast.AssignmentNode, resolvedTypes [][]ast.TypeNode) error {
	for i, lv := range assign.LValues {
		switch l := lv.(type) {
		case ast.VariableNode:
			if err := tc.inferAssignmentVariableLValue(assign, l, i, resolvedTypes); err != nil {
				return err
			}
		case ast.IndexExpressionNode:
			if err := tc.inferAssignmentIndexLValue(assign, l, i, resolvedTypes); err != nil {
				return err
			}
		case ast.DereferenceNode:
			if err := tc.inferAssignmentDerefLValue(l, i, resolvedTypes); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported assignment target type: %T", lv)
		}
	}
	return nil
}

func (tc *TypeChecker) inferAssignmentVariableLValue(assign ast.AssignmentNode, l ast.VariableNode, i int, resolvedTypes [][]ast.TypeNode) error {
	if l.Ident.ID == "_" {
		// Blank identifier discards the RHS; still typecheck via resolvedTypes, do not bind.
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssignmentTypes",
		}).Trace("Skipping binding for blank identifier assignment")
		return nil
	}

	isVarDeclaration := len(assign.ExplicitTypes) > i && assign.ExplicitTypes[i] != nil

	if !assign.IsShort && !isVarDeclaration {
		if err := tc.checkExistingVariableAssignment(assign, l, i, resolvedTypes); err != nil {
			return err
		}
	}

	if isVarDeclaration {
		if err := tc.checkVarDeclarationType(l); err != nil {
			return err
		}
		if err := tc.checkNilVarInitialization(assign, l, i); err != nil {
			return err
		}
	}

	if isVarDeclaration && len(assign.RValues) > 0 && i < len(resolvedTypes) && len(resolvedTypes[i]) == 1 {
		if err := tc.checkVarDeclarationRHS(assign, l, i, resolvedTypes); err != nil {
			return err
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
	return nil
}

func (tc *TypeChecker) checkExistingVariableAssignment(assign ast.AssignmentNode, l ast.VariableNode, i int, resolvedTypes [][]ast.TypeNode) error {
	parts := strings.Split(string(l.Ident.ID), ".")
	if len(parts) > 1 {
		return tc.checkFieldAssignment(l, i, assign, resolvedTypes)
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
	return nil
}

func (tc *TypeChecker) checkFieldAssignment(l ast.VariableNode, i int, assign ast.AssignmentNode, resolvedTypes [][]ast.TypeNode) error {
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
		return nil
	}
	if !tc.IsTypeCompatible(resolvedTypes[i][0], lhsType) {
		return reportf(l.Ident.Span, "assignment-type-mismatch",
			"assignment type mismatch",
			fmt.Sprintf("Cannot assign `%s` to `%s` (expected `%s`).", formatTypeIdentForDiag(resolvedTypes[i][0].Ident), l.Ident.ID, formatTypeIdentForDiag(lhsType.Ident)),
			"convert the value or change the field type")
	}
	tc.storeInferredType(l, []ast.TypeNode{lhsType})
	return nil
}

func (tc *TypeChecker) checkVarDeclarationType(l ast.VariableNode) error {
	explicitType := l.ExplicitType
	isPointer := explicitType.Ident == ast.TypePointer
	isBuiltin := explicitType.Ident == ast.TypeString || explicitType.Ident == ast.TypeInt || explicitType.Ident == ast.TypeFloat || explicitType.Ident == ast.TypeBool || explicitType.Ident == ast.TypeError || explicitType.Ident == ast.TypeVoid || explicitType.Ident == ast.TypeArray || explicitType.Ident == ast.TypeMap || explicitType.Ident == ast.TypeShape || explicitType.Ident == ast.TypeObject
	_, isDefined := tc.Defs[explicitType.Ident]
	if !isPointer && !isBuiltin && !isDefined {
		return reportf(l.Ident.Span, "undefined-type",
			fmt.Sprintf("undefined type `%s`", formatTypeIdentForDiag(explicitType.Ident)),
			fmt.Sprintf("Type name `%s` in the variable declaration is not defined.", formatTypeIdentForDiag(explicitType.Ident)),
			"declare the type or use a built-in name")
	}
	return nil
}

func (tc *TypeChecker) checkNilVarInitialization(assign ast.AssignmentNode, l ast.VariableNode, i int) error {
	if len(assign.RValues) != 1 {
		return nil
	}
	if _, isNil := assign.RValues[0].(ast.NilLiteralNode); !isNil {
		return nil
	}
	explicitType := l.ExplicitType
	if !isNilableAssignmentType(explicitType) {
		return reportf(l.Ident.Span, "nil-assign-type",
			fmt.Sprintf("cannot assign nil to `%s`", formatTypeIdentForDiag(explicitType.Ident)),
			fmt.Sprintf("Type `%s` cannot be initialized with `nil`.", formatTypeIdentForDiag(explicitType.Ident)),
			"use a pointer, map, slice, interface, or func type — or supply a value")
	}
	return nil
}

func isNilableAssignmentType(explicitType ast.TypeNode) bool {
	switch explicitType.Ident {
	case ast.TypePointer, ast.TypeObject, ast.TypeMap, ast.TypeArray, ast.TypeIdent("Func"):
		return true
	default:
		return false
	}
}

func (tc *TypeChecker) checkVarDeclarationRHS(assign ast.AssignmentNode, l ast.VariableNode, i int, resolvedTypes [][]ast.TypeNode) error {
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
			fmt.Sprintf("Cannot assign `%s` to `%s` (expected `%s`).", formatTypeIdentForDiag(rhs.Ident), l.Ident.ID, formatTypeIdentForDiag(lhs.Ident)),
			"convert the initializer or change the declared type")
	}
	return nil
}

func (tc *TypeChecker) inferAssignmentIndexLValue(assign ast.AssignmentNode, l ast.IndexExpressionNode, i int, resolvedTypes [][]ast.TypeNode) error {
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
	return nil
}

func (tc *TypeChecker) inferAssignmentDerefLValue(l ast.DereferenceNode, i int, resolvedTypes [][]ast.TypeNode) error {
	derefSpan := spanOfExpression(l.Value)
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
	return nil
}
