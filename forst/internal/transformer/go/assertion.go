package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

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

	// Emit a Go type definition for value assertions
	if len(assertion.Constraints) == 1 && assertion.Constraints[0].Name == "Value" {
		hashName := string(assertionType.Ident)
		// Only emit if not already present
		if !t.Output.HasType(hashName) {
			decl := &goast.GenDecl{
				Tok: token.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: goast.NewIdent(hashName),
						Type: goast.NewIdent("string"),
					},
				},
			}
			t.Output.AddType(decl)
		}
	}

	var expr goast.Expr = goast.NewIdent(string(assertionType.Ident))
	return &expr, nil
}
