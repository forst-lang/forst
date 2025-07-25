package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	logrus "github.com/sirupsen/logrus"
)

func (t *Transformer) transformAssertionType(assertion *ast.AssertionNode) (*goast.Expr, error) {
	t.log.WithFields(logrus.Fields{
		"function":  "transformAssertionType",
		"assertion": assertion,
	}).Debugf("transformAssertionType")
	assertionType, err := t.TypeChecker.LookupAssertionType(assertion)
	if assertionType == nil {
		t.log.WithFields(logrus.Fields{
			"function":      "transformAssertionType",
			"assertionType": "nil",
		}).Debugf("assertionType: nil")
	} else {
		t.log.WithFields(logrus.Fields{
			"function":      "transformAssertionType",
			"assertionType": *assertionType,
		}).Debugf("assertionType: %s", *assertionType)
	}
	if err != nil {
		err = fmt.Errorf("failed to lookup assertion type during transformation: %w", err)
		t.log.WithFields(logrus.Fields{
			"function": "transformAssertionType",
			"error":    err,
		}).Error("transforming assertion type failed")
		return nil, err
	}

	// Handle value assertions by generating concrete Go types instead of recursive aliases
	constraintName := ""
	if len(assertion.Constraints) > 0 {
		constraintName = assertion.Constraints[0].Name
	}
	t.log.WithFields(logrus.Fields{
		"function":          "transformAssertionType",
		"constraintsLength": len(assertion.Constraints),
		"constraintName":    constraintName,
		"isValueAssertion":  len(assertion.Constraints) == 1 && constraintName == ast.ValueConstraint,
	}).Debugf("Checking value assertion")
	if len(assertion.Constraints) == 1 && constraintName == ast.ValueConstraint {
		// For value assertions, we need to determine the concrete Go type based on the value
		if len(assertion.Constraints[0].Args) > 0 {
			arg := assertion.Constraints[0].Args[0]
			if arg.Value != nil {
				switch (*arg.Value).(type) {
				case ast.StringLiteralNode:
					// String literals should be typed as string
					var expr goast.Expr = goast.NewIdent("string")
					return &expr, nil
				case ast.IntLiteralNode:
					// Int literals should be typed as int
					var expr goast.Expr = goast.NewIdent("int")
					return &expr, nil
				case ast.FloatLiteralNode:
					// Float literals should be typed as float64
					var expr goast.Expr = goast.NewIdent("float64")
					return &expr, nil
				case ast.BoolLiteralNode:
					// Bool literals should be typed as bool
					var expr goast.Expr = goast.NewIdent("bool")
					return &expr, nil
				case ast.VariableNode:
					// Variable references should use the variable's actual type
					variableNode := (*arg.Value).(ast.VariableNode)
					varType, err := t.TypeChecker.LookupVariableType(&variableNode, t.currentScope())
					if err == nil {
						// Use the variable's actual type
						var expr goast.Expr = goast.NewIdent(string(varType.Ident))
						return &expr, nil
					}
					// Fallback to string if variable type not found
					var expr goast.Expr = goast.NewIdent("string")
					return &expr, nil
				case ast.ReferenceNode:
					// Reference expressions (Ref(Variable(x))) should use a pointer to the variable's type
					// For now, assume *string for reference expressions
					starExpr := &goast.StarExpr{X: goast.NewIdent("string")}
					var expr goast.Expr = starExpr
					return &expr, nil
				default:
					// Default to string for unknown value types
					var expr goast.Expr = goast.NewIdent("string")
					return &expr, nil
				}
			}
		}
		// If no value or unknown value type, default to string
		var expr goast.Expr = goast.NewIdent("string")
		return &expr, nil
	}

	// Use original name for user-defined types, hash-based name for structural types
	var name string
	if ast.IsHashBasedType(*assertionType) {
		// For hash-based types, use getTypeAliasNameForTypeNode to get the hash-based name
		name, err = t.getTypeAliasNameForTypeNode(*assertionType)
		if err != nil {
			err = fmt.Errorf("failed to get type alias name for assertion type: %w", err)
			t.log.WithFields(logrus.Fields{
				"function": "transformAssertionType",
				"error":    err,
			}).WithError(err).Error("getting type alias name failed")
			return nil, err
		}
	} else {
		// For user-defined types, use the original name
		name = string(assertionType.Ident)
	}
	var expr goast.Expr = goast.NewIdent(name)
	return &expr, nil
}
