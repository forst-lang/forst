package typechecker

import (
	"forst/internal/ast"

	log "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) storeInferredVariableType(variable ast.VariableNode, typ ast.TypeNode) {
	log.Tracef("Storing inferred variable type for variable %s: %s", variable.Ident.ID, typ)
	tc.storeSymbol(variable.Ident.ID, []ast.TypeNode{typ}, SymbolVariable)
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
	tc.Functions[fn.Ident.ID] = FunctionSignature{
		Ident:       fn.Ident,
		Parameters:  params,
		ReturnTypes: fn.ReturnTypes,
	}

	// Store function symbol
	tc.storeSymbol(fn.Ident.ID, fn.ReturnTypes, SymbolFunction)

	// Store parameter symbols
	for _, param := range fn.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			tc.storeSymbol(p.Ident.ID, []ast.TypeNode{p.Type}, SymbolParameter)
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}
	}
}

func (tc *TypeChecker) registerTypeGuard(guard ast.TypeGuardNode) {
	// Store function signature
	params := make([]ParameterSignature, len(guard.Params))
	for i, param := range guard.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			params[i] = ParameterSignature{
				Ident: p.Ident,
				Type:  p.Type,
			}
		case ast.DestructuredParamNode:
			// TODO: Handle destructured params
			continue
		}
	}

	// tc.Functions[ast.Identifier(guard.Ident)] = FunctionSignature{
	// 	Ident:       ast.Ident{ID: ast.Identifier(guard.Ident)},
	// 	Parameters:  params,
	// 	ReturnTypes: []ast.TypeNode{{Ident: ast.TypeVoid}},
	// }

	// Register type guard in Defs
	if _, exists := tc.Defs[ast.TypeIdent(guard.Ident)]; !exists {
		tc.Defs[ast.TypeIdent(guard.Ident)] = guard
	}

	// Store function symbol
	tc.storeSymbol(guard.Ident, []ast.TypeNode{{Ident: ast.TypeVoid}}, SymbolTypeGuard)

	// Store receiver symbol
	tc.storeSymbol(ast.Identifier(guard.Subject.GetIdent()), guard.Subject.GetType().TypeParams, SymbolParameter)

	// Store parameter symbols
	for _, param := range guard.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			tc.storeSymbol(p.Ident.ID, []ast.TypeNode{p.Type}, SymbolParameter)
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}
	}
}
