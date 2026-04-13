package typechecker

import "forst/internal/ast"

func (tc *TypeChecker) inferEnsureNode(ensureNode ast.EnsureNode) ([]ast.TypeNode, error) {
	variableType, err := tc.inferEnsureType(ensureNode)
	if err != nil {
		return nil, err
	}

	if ensureNode.Block != nil {
		tc.pushScope(ensureNode.Block)
		if _, err := tc.inferExpressionType(ensureNode.Variable); err != nil {
			return nil, err
		}
		if tc.ensureUsesBuiltinResultOkErrDiscriminator(ensureNode) {
			tc.applyEnsureBlockResultFailureNarrowing(ensureNode)
		} else {
			tc.applyEnsureSuccessorNarrowing(ensureNode)
		}
		_, err = tc.inferNodeTypes(ensureNode.Block.Body, ensureNode.Block)
		tc.popScope()
		if err != nil {
			return nil, err
		}
		if tc.ensureUsesBuiltinResultOkErrDiscriminator(ensureNode) {
			tc.applyEnsureSuccessorNarrowing(ensureNode)
		}
	} else {
		if _, err := tc.inferExpressionType(ensureNode.Variable); err != nil {
			return nil, err
		}
		tc.applyEnsureSuccessorNarrowing(ensureNode)
	}

	tc.storeInferredType(ensureNode.Assertion, []ast.TypeNode{variableType})
	return nil, nil
}
