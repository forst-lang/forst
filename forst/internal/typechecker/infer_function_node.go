package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferFunctionNode(node ast.Node) ([]ast.TypeNode, error) {
	var functionNode ast.FunctionNode
	switch n := node.(type) {
	case ast.FunctionNode:
		functionNode = n
	case *ast.FunctionNode:
		if n == nil {
			return nil, nil
		}
		functionNode = *n
	default:
		return nil, fmt.Errorf("inferFunctionNode: unexpected node type %T", node)
	}
	prevFn := tc.currentFunction
	tc.currentFunction = &functionNode
	prevErrBranchDepth := tc.resultErrIfBranchDepth
	tc.resultErrIfBranchDepth = 0
	restoreLabels := tc.pushLoopLabelScope()
	defer func() {
		tc.currentFunction = prevFn
		tc.resultErrIfBranchDepth = prevErrBranchDepth
		restoreLabels()
	}()

	tc.log.WithFields(logrus.Fields{
		"function": "inferNodeType",
		"fn":       functionNode.Ident.ID,
		"phase":    "ENTER",
	}).Debug("Function node type inference")

	if err := tc.RestoreScope(node); err != nil {
		return nil, err
	}
	tc.log.WithFields(logrus.Fields{
		"function": "inferNodeType",
		"fn":       functionNode.Ident.ID,
	}).Debug("Restored function scope")

	for i, param := range functionNode.Params {
		switch typedParam := param.(type) {
		case ast.SimpleParamNode:
			paramType := typedParam.Type
			if sig, ok := tc.Functions[functionNode.Ident.ID]; ok && i < len(sig.Parameters) {
				paramType = sig.Parameters[i].Type
			}
			tc.scopeStack.currentScope().RegisterSymbol(
				typedParam.Ident.ID,
				[]ast.TypeNode{paramType},
				SymbolVariable)
			tc.bindVariableGoTypeFromParamType(typedParam.Ident.ID, paramType)
		case ast.DestructuredParamNode:
			paramType := typedParam.Type
			if sig, ok := tc.Functions[functionNode.Ident.ID]; ok && i < len(sig.Parameters) {
				paramType = sig.Parameters[i].Type
			}
			tc.registerDestructuredParamSymbols(typedParam.Fields, paramType, SymbolVariable)
		}
	}
	tc.DebugPrintCurrentScope()

	params := make([]ast.Node, len(functionNode.Params))
	for index, param := range functionNode.Params {
		params[index] = param
	}

	paramTypes, err := tc.inferNodeTypes(params, node)
	if err != nil {
		return nil, err
	}

	sig, hasSig := tc.Functions[functionNode.Ident.ID]
	for index, inferredParamTypes := range paramTypes {
		if hasSig && len(sig.TypeParams) > 0 {
			continue
		}
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

	if signature, ok := tc.Functions[functionNode.Ident.ID]; ok && len(signature.TypeParams) == 0 {
		for index := range signature.Parameters {
			if index < len(paramTypes) && len(paramTypes[index]) >= 1 {
				signature.Parameters[index].Type = paramTypes[index][0]
			}
		}
		tc.Functions[functionNode.Ident.ID] = signature
	}

	for _, bodyNode := range functionNode.Body {
		if _, err := tc.inferNodeType(bodyNode); err != nil {
			return nil, err
		}
	}
	if err := tc.checkFunctionLabels(functionNode.Body); err != nil {
		return nil, err
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
