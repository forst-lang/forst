package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferNodeType(node ast.Node) ([]ast.TypeNode, error) {
	if tc.log.IsLevelEnabled(logrus.TraceLevel) {
		tc.log.WithFields(logrus.Fields{
			"node":     node.String(),
			"function": "inferNodeType",
		}).Trace("Inferring node type")
	}

	handlers := []func(ast.Node) ([]ast.TypeNode, bool, error){
		tc.inferNodeTypeLeaf,
		tc.inferNodeTypeParams,
		tc.inferNodeTypeExpression,
		tc.inferNodeTypeDecl,
		tc.inferNodeTypeControl,
		tc.inferNodeTypeDeferGo,
		tc.inferNodeTypeElseBlock,
		tc.inferNodeTypeFunction,
		tc.inferNodeTypeEnsure,
		tc.inferNodeTypeReturn,
		tc.inferNodeTypeTypeGuard,
	}
	for _, h := range handlers {
		types, ok, err := h(node)
		if ok {
			return types, err
		}
	}
	return nil, fmt.Errorf("%s", typecheckErrorMessageWithNode(&node, fmt.Sprintf("unsupported node type %T", node)))
}
