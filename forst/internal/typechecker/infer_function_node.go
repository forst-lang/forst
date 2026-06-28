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
			tc.registerDestructuredParamSymbols(typedParam.Fields, typedParam.Type, SymbolVariable)
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

		switch p := param.(type) {
		case ast.SimpleParamNode:
			tc.scopeStack.currentScope().RegisterSymbol(
				p.Ident.ID,
				inferredParamTypes,
				SymbolVariable)
			if len(inferredParamTypes) > 0 {
				tc.VariableTypes[p.Ident.ID] = append([]ast.TypeNode(nil), inferredParamTypes...)
			}
		case ast.DestructuredParamNode:
			if shapeFields, ok := tc.ShapeFieldsFromParamType(p.Type); ok {
				for _, fieldName := range p.Fields {
					sf, ok := shapeFields[fieldName]
					if !ok {
						continue
					}
					if tn, ok := ShapeFieldTypeNode(sf); ok {
						fieldTypes := []ast.TypeNode{tn}
						tc.scopeStack.currentScope().RegisterSymbol(
							ast.Identifier(fieldName),
							fieldTypes,
							SymbolVariable)
						tc.VariableTypes[ast.Identifier(fieldName)] = append([]ast.TypeNode(nil), fieldTypes...)
					}
				}
			}
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
	if err := tc.validateInferredReceiverMethodReturn(functionNode, inferredType); err != nil {
		return nil, err
	}
	tc.popScope()

	tc.log.WithFields(logrus.Fields{
		"function": "inferNodeType",
		"fn":       functionNode.Ident.ID,
		"phase":    "EXIT",
	}).Debug("Function node type inference")

	return inferredType, nil
}
