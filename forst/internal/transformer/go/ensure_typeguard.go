package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	"github.com/sirupsen/logrus"
)

// transformTypeGuardEnsure transforms a type guard ensure
func (t *Transformer) transformTypeGuardEnsure(ensure *ast.EnsureNode) ([]goast.Stmt, error) {
	// Get the variable type from the symbol table
	varType, err := t.TypeChecker.LookupVariableType(&ensure.Variable, t.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup variable type: %w", err)
	}

	// Transform each constraint
	var transformedStmts []goast.Stmt
	for _, constraint := range ensure.Assertion.Constraints {
		transformed, err := t.transformEnsureConstraint(*ensure, constraint, varType)
		if err != nil {
			return nil, fmt.Errorf("failed to transform constraint: %w", err)
		}
		transformedStmts = append(transformedStmts, &goast.ExprStmt{X: transformed})
	}

	return transformedStmts, nil
}

// Helper to look up a TypeGuardNode by name
func (t *Transformer) lookupTypeGuardNode(name string) (*ast.TypeGuardNode, error) {
	for _, def := range t.TypeChecker.Defs {
		if tg, ok := def.(ast.TypeGuardNode); ok {
			t.log.WithFields(logrus.Fields{
				"requested": name,
				"candidate": tg.GetIdent(),
				"function":  "lookupTypeGuardNode",
			}).Trace("candidate check (value)")
			if tg.GetIdent() == name {
				t.log.WithFields(logrus.Fields{
					"requested": name,
					"found":     true,
				}).Trace("lookupTypeGuardNode: found match (value)")
				return &tg, nil
			}
		}
		if tgp, ok := def.(*ast.TypeGuardNode); ok {
			t.log.WithFields(logrus.Fields{
				"requested": name,
				"candidate": tgp.GetIdent(),
				"function":  "lookupTypeGuardNode",
			}).Trace("candidate check (pointer)")
			if tgp.GetIdent() == name {
				t.log.WithFields(logrus.Fields{
					"requested": name,
					"found":     true,
					"function":  "lookupTypeGuardNode",
				}).Trace("found match (pointer)")
				return tgp, nil
			}
		}
	}

	t.log.WithFields(logrus.Fields{
		"requested": name,
		"found":     false,
		"function":  "lookupTypeGuardNode",
	}).Trace("not found")
	return nil, fmt.Errorf("type guard not found: %s", name)
}

func (t *Transformer) isTypeGuardCompatible(varType ast.TypeNode, typeGuard *ast.TypeGuardNode) bool {
	t.log.WithFields(logrus.Fields{
		"varType":   varType.Ident,
		"typeGuard": typeGuard.GetIdent(),
		"function":  "isTypeGuardCompatible",
	}).Trace("Checking type guard compatibility")

	// Use varType directly as the base type
	baseType := varType
	t.log.WithFields(logrus.Fields{
		"baseType": baseType.Ident,
		"function": "isTypeGuardCompatible",
	}).Trace("Expected base type of type guard identified based on variable type")

	// Check if the type guard is defined for the base type
	for _, param := range typeGuard.Parameters() {
		paramType := param.GetType()
		t.log.WithFields(logrus.Fields{
			"paramType": paramType.Ident,
			"baseType":  baseType.Ident,
			"function":  "isTypeGuardCompatible",
		}).Trace("Checking parameter type")

		// Use the type checker's IsTypeCompatible function to handle type aliases and structural compatibility
		if t.TypeChecker.IsTypeCompatible(baseType, paramType) {
			t.log.WithFields(logrus.Fields{
				"typeGuard": typeGuard.GetIdent(),
				"baseType":  baseType.Ident,
				"paramType": paramType.Ident,
				"function":  "isTypeGuardCompatible",
			}).Trace("Found compatible type guard")
			return true
		}
	}

	t.log.WithFields(logrus.Fields{
		"typeGuard": typeGuard.GetIdent(),
		"baseType":  baseType.Ident,
		"function":  "isTypeGuardCompatible",
	}).Trace("No compatible type guard found")
	return false
}
