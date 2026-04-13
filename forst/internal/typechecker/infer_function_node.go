package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferFunctionNode(functionNode ast.FunctionNode) ([]ast.TypeNode, error) {
	prevFn := tc.currentFunction
	tc.currentFunction = &functionNode
	prevErrBranchDepth := tc.resultErrIfBranchDepth
	tc.resultErrIfBranchDepth = 0
	defer func() {
		tc.currentFunction = prevFn
		tc.resultErrIfBranchDepth = prevErrBranchDepth
	}()

	tc.log.WithFields(logrus.Fields{
		"function": "inferNodeType",
		"fn":       functionNode.Ident.ID,
		"phase":    "ENTER",
	}).Debug("Function node type inference")

	if err := tc.RestoreScope(functionNode); err != nil {
		return nil, err
	}
	tc.log.WithFields(logrus.Fields{
		"function": "inferNodeType",
		"fn":       functionNode.Ident.ID,
	}).Debug("Restored function scope")

	for _, param := range functionNode.Params {
		switch typedParam := param.(type) {
		case ast.SimpleParamNode:
			tc.scopeStack.currentScope().RegisterSymbol(
				typedParam.Ident.ID,
				[]ast.TypeNode{typedParam.Type},
				SymbolVariable)
		case ast.DestructuredParamNode:
			continue
		}
	}
	tc.DebugPrintCurrentScope()

	params := make([]ast.Node, len(functionNode.Params))
	for index, param := range functionNode.Params {
		params[index] = param
	}

	paramTypes, err := tc.inferNodeTypes(params, functionNode)
	if err != nil {
		return nil, err
	}

	for index, inferredParamTypes := range paramTypes {
		param := functionNode.Params[index]
		tc.log.WithFields(logrus.Fields{
			"paramTypes": inferredParamTypes,
			"param":      param.GetIdent(),
			"function":   "inferNodeType",
		}).Trace("Storing param variable type")

		tc.scopeStack.currentScope().RegisterSymbol(
			ast.Identifier(param.GetIdent()),
			inferredParamTypes,
			SymbolVariable)

		if simpleParam, ok := param.(ast.SimpleParamNode); ok && len(inferredParamTypes) > 0 {
			tc.VariableTypes[simpleParam.Ident.ID] = append([]ast.TypeNode(nil), inferredParamTypes...)
		}
	}

	if signature, ok := tc.Functions[functionNode.Ident.ID]; ok {
		for index := range signature.Parameters {
			if index < len(paramTypes) && len(paramTypes[index]) >= 1 {
				signature.Parameters[index].Type = paramTypes[index][0]
			}
		}
	}

	for _, bodyNode := range functionNode.Body {
		if _, err := tc.inferNodeType(bodyNode); err != nil {
			return nil, err
		}
	}

	inferredType, err := tc.inferFunctionReturnType(functionNode)
	if err != nil {
		return nil, err
	}
	tc.storeInferredFunctionReturnType(&functionNode, inferredType)
	tc.popScope()

	tc.log.WithFields(logrus.Fields{
		"function": "inferNodeType",
		"fn":       functionNode.Ident.ID,
		"phase":    "EXIT",
	}).Debug("Function node type inference")

	return inferredType, nil
}
