package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"strings"

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
	t.log.WithFields(logrus.Fields{
		"requested": name,
		"function":  "lookupTypeGuardNode",
	}).Info("Starting lookup for type guard")

	for _, def := range t.TypeChecker.Defs {
		if tg, ok := def.(ast.TypeGuardNode); ok {
			t.log.WithFields(logrus.Fields{
				"requested": name,
				"candidate": tg.GetIdent(),
				"function":  "lookupTypeGuardNode",
			}).Info("candidate check (value)")
			if tg.GetIdent() == name {
				t.log.WithFields(logrus.Fields{
					"requested": name,
					"found":     true,
				}).Info("lookupTypeGuardNode: found match (value)")
				return &tg, nil
			}
		}
		if tgp, ok := def.(*ast.TypeGuardNode); ok {
			t.log.WithFields(logrus.Fields{
				"requested": name,
				"candidate": tgp.GetIdent(),
				"function":  "lookupTypeGuardNode",
			}).Info("candidate check (pointer)")
			if tgp.GetIdent() == name {
				t.log.WithFields(logrus.Fields{
					"requested": name,
					"found":     true,
					"function":  "lookupTypeGuardNode",
				}).Info("found match (pointer)")
				return tgp, nil
			}
		}
	}

	t.log.WithFields(logrus.Fields{
		"requested": name,
		"found":     false,
		"function":  "lookupTypeGuardNode",
	}).Info("not found")
	return nil, fmt.Errorf("type guard not found: %s", name)
}

func (t *Transformer) isTypeGuardCompatible(varType ast.TypeNode, typeGuard *ast.TypeGuardNode) bool {
	t.log.WithFields(logrus.Fields{
		"varType":   varType.Ident,
		"typeGuard": typeGuard.GetIdent(),
		"function":  "isTypeGuardCompatible",
	}).Info("Checking type guard compatibility")

	// Use varType directly as the base type
	baseType := varType
	t.log.WithFields(logrus.Fields{
		"baseType": baseType.Ident,
		"function": "isTypeGuardCompatible",
	}).Info("Expected base type of type guard identified based on variable type")

	// Check if the type guard is defined for the base type
	for _, param := range typeGuard.Parameters() {
		paramType := param.GetType()
		t.log.WithFields(logrus.Fields{
			"paramType": paramType.Ident,
			"baseType":  baseType.Ident,
			"function":  "isTypeGuardCompatible",
		}).Info("Checking parameter type")

		// Use the type checker's IsTypeCompatible function to handle type aliases and structural compatibility
		compatible := t.TypeChecker.IsTypeCompatible(baseType, paramType)
		t.log.WithFields(logrus.Fields{
			"typeGuard":  typeGuard.GetIdent(),
			"baseType":   baseType.Ident,
			"paramType":  paramType.Ident,
			"compatible": compatible,
			"function":   "isTypeGuardCompatible",
		}).Info("Type compatibility check result")

		if compatible {
			t.log.WithFields(logrus.Fields{
				"typeGuard": typeGuard.GetIdent(),
				"baseType":  baseType.Ident,
				"paramType": paramType.Ident,
				"function":  "isTypeGuardCompatible",
			}).Info("Found compatible type guard")
			return true
		}

		// If the base type is a hash-based type (T_*) and the param type is a named type,
		// check if they are structurally compatible by looking up their definitions
		if strings.HasPrefix(string(baseType.Ident), "T_") && !strings.HasPrefix(string(paramType.Ident), "T_") {
			// Get the base type definition
			if baseDef, exists := t.TypeChecker.Defs[baseType.Ident]; exists {
				if baseTypeDef, ok := baseDef.(ast.TypeDefNode); ok {
					if baseShapeExpr, ok := baseTypeDef.Expr.(ast.TypeDefShapeExpr); ok {
						// Get the param type definition
						if paramDef, exists := t.TypeChecker.Defs[paramType.Ident]; exists {
							if paramTypeDef, ok := paramDef.(ast.TypeDefNode); ok {
								if paramShapeExpr, ok := paramTypeDef.Expr.(ast.TypeDefShapeExpr); ok {
									// Check if the shapes are structurally compatible
									if t.shapesCompatibleForExpectedType(&baseShapeExpr.Shape, &paramShapeExpr.Shape) {
										t.log.WithFields(logrus.Fields{
											"typeGuard": typeGuard.GetIdent(),
											"baseType":  baseType.Ident,
											"paramType": paramType.Ident,
											"function":  "isTypeGuardCompatible",
										}).Info("Found structurally compatible type guard")
										return true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	t.log.WithFields(logrus.Fields{
		"typeGuard": typeGuard.GetIdent(),
		"baseType":  baseType.Ident,
		"function":  "isTypeGuardCompatible",
	}).Info("No compatible type guard found")
	return false
}
