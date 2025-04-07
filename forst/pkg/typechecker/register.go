package typechecker

import (
	"forst/pkg/ast"
)

func (tc *TypeChecker) storeInferredVariableType(variable ast.VariableNode, typ ast.TypeNode) {
	tc.storeSymbol(variable.Ident.Id, []ast.TypeNode{typ}, SymbolVariable)
	tc.storeInferredType(variable, []ast.TypeNode{typ})
}

func (tc *TypeChecker) registerFunction(fn ast.FunctionNode) {
	// Store function signature
	params := make([]ParameterSignature, len(fn.Params))
	for i, param := range fn.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			params[i] = ParameterSignature{
				Ident: p.Ident,
				Type:  p.Type,
			}
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}
	}
	tc.Functions[fn.Id()] = FunctionSignature{
		Ident:       fn.Ident,
		Parameters:  params,
		ReturnTypes: fn.ReturnTypes,
	}

	// Store function symbol
	tc.storeSymbol(fn.Ident.Id, fn.ReturnTypes, SymbolFunction)

	// Store parameter symbols
	for _, param := range fn.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			tc.storeSymbol(p.Ident.Id, []ast.TypeNode{p.Type}, SymbolParameter)
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}
	}
}
