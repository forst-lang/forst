package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// ensureTypeKind ensures that a TypeNode has the correct TypeKind set
func ensureTypeKind(typeNode ast.TypeNode, expectedKind ast.TypeKind) ast.TypeNode {
	if typeNode.TypeKind != expectedKind {
		typeNode.TypeKind = expectedKind
	}
	return typeNode
}

// ensureUserDefinedType ensures that a TypeNode is marked as user-defined
func ensureUserDefinedType(typeNode ast.TypeNode) ast.TypeNode {
	return ensureTypeKind(typeNode, ast.TypeKindUserDefined)
}

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
		"typeKind": "user-defined", // All registered types are user-defined
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
	// Ensure that user-defined types have the correct TypeKind
	// Update the shape fields to have correct TypeKind
	for fieldName, field := range shape.Fields {
		if field.Type != nil {
			// Ensure user-defined field types have correct TypeKind
			if field.Type.TypeKind != ast.TypeKindHashBased && !tc.isBuiltinType(field.Type.Ident) {
				field.Type.TypeKind = ast.TypeKindUserDefined
			}
			shape.Fields[fieldName] = field
		}
	}

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
		"typeKind": "user-defined", // All registered types are user-defined
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

	// Ensure return types have correct TypeKind
	processedReturnTypes := make([]ast.TypeNode, len(fn.ReturnTypes))
	for i, returnType := range fn.ReturnTypes {
		// For user-defined types, ensure they're marked as user-defined
		if returnType.TypeKind != ast.TypeKindHashBased && !tc.isBuiltinType(returnType.Ident) {
			processedReturnTypes[i] = ensureUserDefinedType(returnType)
		} else {
			processedReturnTypes[i] = returnType
		}
	}

	tc.Functions[fn.Ident.ID] = FunctionSignature{
		Ident:       fn.Ident,
		Parameters:  params,
		ReturnTypes: processedReturnTypes,
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
