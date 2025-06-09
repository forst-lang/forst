package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) storeInferredVariableType(variable ast.VariableNode, typ []ast.TypeNode) {
	tc.log.Tracef("Storing inferred variable type for variable %s: %s", variable.Ident.ID, typ)
	tc.storeSymbol(variable.Ident.ID, typ, SymbolVariable)
	tc.storeInferredType(variable, typ)
}

// Stores a type definition that will be used by code generators
// to create corresponding type definitions in the target language.
// For example, a Forst type definition like `type PhoneNumber = String.Min(3)`
// may be transformed into a TypeScript type with validation decorators.
func (tc *TypeChecker) registerType(node ast.TypeDefNode) {
	if _, exists := tc.Defs[node.Ident]; exists {
		return
	}
	// Store the type definition node
	tc.Defs[node.Ident] = node
	tc.log.WithFields(logrus.Fields{
		"node":     node.String(),
		"function": "registerType",
	}).Trace("Registered type")

	// If this is a shape type, also store the underlying ShapeNode for field access
	if assertionExpr, ok := node.Expr.(ast.TypeDefAssertionExpr); ok {
		if assertionExpr.Assertion != nil {
			// If this is a direct shape alias (e.g. type AppContext = { ... })
			if assertionExpr.Assertion.BaseType != nil && *assertionExpr.Assertion.BaseType == ast.TypeShape {
				// If there are no constraints, check if the assertion has a Match constraint with a shape
				if len(assertionExpr.Assertion.Constraints) == 0 && assertionExpr.Assertion.BaseType != nil {
					// See if the assertion itself has a shape (Match constraint)
					if assertionExpr.Assertion != nil && assertionExpr.Assertion.Constraints != nil {
						for _, constraint := range assertionExpr.Assertion.Constraints {
							for _, arg := range constraint.Args {
								if arg.Shape != nil {
									tc.log.WithFields(logrus.Fields{
										"node":     node.String(),
										"function": "registerType",
										"shape":    arg.Shape,
									}).Trace("Registering shape type from assertion")
									tc.registerShapeType(node.Ident, *arg.Shape)
								}
							}
						}
					}
				}
				// Try to extract from constraints if present (existing logic)
				for _, constraint := range assertionExpr.Assertion.Constraints {
					for _, arg := range constraint.Args {
						if arg.Shape != nil {
							tc.log.WithFields(logrus.Fields{
								"node":     node.String(),
								"function": "registerType",
								"shape":    arg.Shape,
							}).Trace("Registering shape type from constraints")
							tc.registerShapeType(node.Ident, *arg.Shape)
						}
					}
				}
			}
		}
	} else if shapeExpr, ok := node.Expr.(ast.TypeDefShapeExpr); ok {
		// If the type definition is directly a shape, store it with a special key
		tc.log.WithFields(logrus.Fields{
			"node":     node.String(),
			"function": "registerType",
			"shape":    shapeExpr.Shape,
		}).Trace("Registering shape type from type definition")

		tc.registerShapeType(node.Ident, shapeExpr.Shape)
	}
}

// registerShapeType registers a shape type with its fields
func (tc *TypeChecker) registerShapeType(ident ast.TypeIdent, shape ast.ShapeNode) {
	tc.Defs[ident] = ast.TypeDefNode{
		Ident: ident,
		Expr: ast.TypeDefShapeExpr{
			Shape: shape,
		},
	}

	tc.log.WithFields(logrus.Fields{
		"ident":    ident,
		"shape":    shape,
		"function": "registerShapeType",
	}).Trace("Registered shape type")
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

func (tc *TypeChecker) registerTypeGuard(guard *ast.TypeGuardNode) {
	// Store type guard in Defs
	if _, exists := tc.Defs[ast.TypeIdent(guard.Ident)]; !exists {
		tc.Defs[ast.TypeIdent(guard.Ident)] = guard
		tc.log.WithFields(logrus.Fields{
			"guard":    guard.Ident,
			"function": "registerTypeGuard",
		}).Trace("Registered type guard")
	}

	// Store subject symbol in the current scope
	tc.storeSymbol(
		ast.Identifier(guard.Subject.GetIdent()),
		[]ast.TypeNode{guard.Subject.GetType()},
		SymbolParameter,
	)

	// Store parameter symbols in the current scope
	for _, param := range guard.Params {
		switch p := param.(type) {
		case ast.SimpleParamNode:
			tc.storeSymbol(
				p.Ident.ID,
				[]ast.TypeNode{p.Type},
				SymbolParameter,
			)
		case ast.DestructuredParamNode:
			// Handle destructured params if needed
			continue
		}
	}
}
