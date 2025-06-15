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
	}).Tracef("transformAssertionType")
	assertionType, err := t.TypeChecker.LookupAssertionType(assertion)
	if assertionType == nil {
		t.log.WithFields(logrus.Fields{
			"function":      "transformAssertionType",
			"assertionType": "nil",
		}).Tracef("assertionType: nil")
	} else {
		t.log.WithFields(logrus.Fields{
			"function":      "transformAssertionType",
			"assertionType": *assertionType,
		}).Tracef("assertionType: %s", *assertionType)
	}
	if err != nil {
		err = fmt.Errorf("failed to lookup assertion type during transformation: %w", err)
		t.log.WithFields(logrus.Fields{
			"function": "transformAssertionType",
			"error":    err,
		}).Error("transforming assertion type failed")
		return nil, err
	}
	var expr goast.Expr = goast.NewIdent(string(assertionType.Ident))
	return &expr, nil
}
