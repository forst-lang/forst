package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	log "github.com/sirupsen/logrus"
)

func (t *Transformer) transformAssertionType(assertion *ast.AssertionNode) (*goast.Expr, error) {
	log.Tracef("transformAssertionType, assertion: %s", *assertion)
	assertionType, err := t.TypeChecker.LookupAssertionType(assertion)
	if assertionType == nil {
		log.Trace("assertionType: nil")
	} else {
		log.Trace(fmt.Sprintf("assertionType: %s", *assertionType))
	}
	if err != nil {
		err = fmt.Errorf("failed to lookup assertion type during transformation: %w", err)
		log.WithError(err).Error("transforming assertion type failed")
		return nil, err
	}
	var expr goast.Expr = goast.NewIdent(string(assertionType.Ident))
	return &expr, nil
}
