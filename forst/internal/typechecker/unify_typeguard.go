package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// validateTypeDefAssertion validates a TypeDefAssertionExpr against the left-hand side type
func (tc *TypeChecker) validateTypeDefAssertion(assertionNode *ast.AssertionNode, varLeftType ast.TypeNode) error {
	if assertionNode == nil {
		return fmt.Errorf("right-hand side of 'is' must be an assertion")
	}

	// Check that the assertion's base type matches the left-hand side type or is a subtype
	if assertionNode.BaseType != nil {
		baseType := ast.TypeNode{Ident: *assertionNode.BaseType}
		if !tc.IsTypeCompatible(varLeftType, baseType) {
			return fmt.Errorf("assertion base type %s is not compatible with left-hand side type %s", baseType.Ident, varLeftType.Ident)
		}
	}

	// Process type guard constraints
	for _, constraint := range assertionNode.Constraints {
		if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
			if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
				// Check that the leftmost variable's type matches the guard's subject type
				subjectType := guardNode.Subject.GetType()
				if !tc.IsTypeCompatible(varLeftType, subjectType) {
					return fmt.Errorf("type guard '%s' requires subject type %s, but got %s",
						constraint.Name, subjectType.Ident, varLeftType.Ident)
				}
			}
		}
	}
	return nil
}

// processTypeGuardFields processes type guard constraints and adds their fields to the shape
func (tc *TypeChecker) processTypeGuardFields(shapeNode *ast.ShapeNode, assertionNode *ast.AssertionNode) {
	if assertionNode == nil {
		return
	}
	tc.log.WithFields(logrus.Fields{
		"function":         "processTypeGuardFields",
		"constraintsCount": len(assertionNode.Constraints),
	}).Tracef("Processing type guard application")
	// Add fields from type guards to the right-hand shape
	for _, constraint := range assertionNode.Constraints {
		tc.log.WithFields(logrus.Fields{
			"function":   "processTypeGuardFields",
			"constraint": constraint.Name,
		}).Tracef("Processing type guard constraint")
		// Look up the type guard definition
		if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
			tc.log.WithFields(logrus.Fields{
				"function":   "processTypeGuardFields",
				"constraint": constraint.Name,
			}).Tracef("Found type guard definition")
			if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
				tc.log.WithFields(logrus.Fields{
					"function":    "processTypeGuardFields",
					"constraint":  constraint.Name,
					"paramsCount": len(guardNode.Params),
				}).Tracef("Found type guard node")

				// Get the parameter name and type from the type guard
				if len(guardNode.Params) > 0 && len(constraint.Args) > 0 {
					param := guardNode.Params[0]
					paramName := param.GetIdent()
					// Use the actual argument type from the constraint application
					argType := constraint.Args[0]
					// Add the new field to the right-hand shape
					if argType.Type != nil {
						shapeNode.Fields[paramName] = ast.ShapeFieldNode{
							Assertion: &ast.AssertionNode{
								BaseType: &argType.Type.Ident,
							},
						}
						tc.log.WithFields(logrus.Fields{
							"function":   "processTypeGuardFields",
							"constraint": constraint.Name,
							"paramName":  paramName,
							"argType":    argType.Type.Ident,
						}).Tracef("Added field to result shape")
					} else {
						tc.log.WithFields(logrus.Fields{
							"function":   "processTypeGuardFields",
							"constraint": constraint.Name,
							"paramName":  paramName,
						}).Errorf("Constraint argument for field has no Type")
					}
				} else {
					tc.log.WithFields(logrus.Fields{
						"function":    "processTypeGuardFields",
						"constraint":  constraint.Name,
						"paramsCount": len(guardNode.Params),
						"argsCount":   len(constraint.Args),
					}).Tracef("Type guard has insufficient params/args")
				}
			} else {
				tc.log.WithFields(logrus.Fields{
					"function":   "processTypeGuardFields",
					"constraint": constraint.Name,
					"guardDef":   guardDef,
				}).Tracef("Definition is not a TypeGuardNode")
			}
		} else {
			tc.log.WithFields(logrus.Fields{
				"function":   "processTypeGuardFields",
				"constraint": constraint.Name,
			}).Tracef("No definition found for type guard")
		}
	}
}

// validateAssertionNode validates a direct assertion node
func (tc *TypeChecker) validateAssertionNode(assertionNode ast.AssertionNode, varLeftType ast.TypeNode) error {
	if len(assertionNode.Constraints) == 1 && assertionNode.BaseType == nil {
		c := assertionNode.Constraints[0]
		if c.Name == "Ok" || c.Name == "Err" {
			if varLeftType.IsResultType() {
				return tc.validateResultDiscriminatorAssertion(assertionNode, varLeftType)
			}
			// Otherwise only valid if a user-defined type guard uses this name (e.g. `is (v N) Ok()`).
			if _, hasGuard := tc.Defs[ast.TypeIdent(c.Name)]; !hasGuard {
				return fmt.Errorf("%s assertion requires Result subject, got %s", c.Name, varLeftType.String())
			}
		}
	}
	for _, constraint := range assertionNode.Constraints {
		if constraint.Name == "Present" {
			// Check if left type is a pointer type
			if varLeftType.Ident != ast.TypePointer {
				return fmt.Errorf("present assertion requires a pointer type, got %s", varLeftType.Ident)
			}
		} else {
			// Check type guard subject type for other constraints
			if guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]; exists {
				if guardNode, ok := guardDef.(ast.TypeGuardNode); ok {
					subjectType := guardNode.Subject.GetType()
					if !tc.IsTypeCompatible(varLeftType, subjectType) {
						return fmt.Errorf("type guard '%s' requires subject type %s, but got %s",
							constraint.Name, subjectType.Ident, varLeftType.Ident)
					}
				}
			}
		}
	}
	return nil
}

// validateResultDiscriminatorAssertion validates `x is Ok(...)` / `Err(...)` when the subject is Result(S,F).
func (tc *TypeChecker) validateResultDiscriminatorAssertion(a ast.AssertionNode, varLeftType ast.TypeNode) error {
	if !varLeftType.IsResultType() || len(varLeftType.TypeParams) < 2 {
		c := "Ok"
		if len(a.Constraints) == 1 && a.Constraints[0].Name == "Err" {
			c = "Err"
		}
		return fmt.Errorf("%s assertion requires Result subject, got %s", c, varLeftType.String())
	}
	if a.BaseType != nil {
		return fmt.Errorf("Ok/Err discriminator cannot use a base type")
	}
	if len(a.Constraints) != 1 {
		return fmt.Errorf("Ok/Err assertion requires exactly one constraint")
	}
	c := a.Constraints[0]
	succ := varLeftType.TypeParams[0]
	fail := varLeftType.TypeParams[1]
	switch c.Name {
	case "Ok":
		switch len(c.Args) {
		case 0:
			return nil
		case 1:
			if c.Args[0].Value == nil {
				return fmt.Errorf("Ok(...) requires a value argument")
			}
			vt, err := tc.inferExpressionType(*c.Args[0].Value)
			if err != nil {
				return err
			}
			if len(vt) != 1 {
				return fmt.Errorf("ok argument: expected a single type")
			}
			if !tc.IsTypeCompatible(vt[0], succ) {
				return fmt.Errorf("Ok(...) value incompatible with success type %s", succ.String())
			}
			return nil
		default:
			return fmt.Errorf("Ok(...) expects at most one argument")
		}
	case "Err":
		switch len(c.Args) {
		case 0:
			return nil
		case 1:
			if c.Args[0].Value == nil {
				return fmt.Errorf("Err(...) requires a value argument")
			}
			vt, err := tc.inferExpressionType(*c.Args[0].Value)
			if err != nil {
				return err
			}
			if len(vt) != 1 {
				return fmt.Errorf("err argument: expected a single type")
			}
			if !tc.IsTypeCompatible(vt[0], fail) {
				return fmt.Errorf("Err(...) value incompatible with failure type %s", fail.String())
			}
			return nil
		default:
			return fmt.Errorf("Err(...) expects at most one argument")
		}
	default:
		return fmt.Errorf("internal: not Ok/Err discriminator")
	}
}
