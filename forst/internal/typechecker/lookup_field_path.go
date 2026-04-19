package typechecker

import (
	"fmt"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// resolveTypeAliasChain follows type aliases until it reaches a non-alias (base) type
func (tc *TypeChecker) resolveTypeAliasChain(typeNode ast.TypeNode) ast.TypeNode {
	visited := map[ast.TypeIdent]bool{}
	current := typeNode
	for {
		def, exists := tc.Defs[current.Ident]
		if !exists {
			break
		}
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			break
		}
		if _, ok := ast.PayloadShape(typeDef.Expr); ok {
			// Ordinary shape typedef or nominal error — resolved name is the struct/error type
			return current
		}
		switch expr := typeDef.Expr.(type) {
		case ast.TypeDefAssertionExpr:
			// Alias to another type (e.g. type Foo = Bar)
			if expr.Assertion != nil && expr.Assertion.BaseType != nil {
				if visited[*expr.Assertion.BaseType] {
					break // cycle
				}
				visited[*expr.Assertion.BaseType] = true
				current = ast.TypeNode{Ident: *expr.Assertion.BaseType}
				continue
			}
			// If no BaseType, treat as non-alias
			return current
		}
		break
	}
	return current
}

// lookupFieldPath recursively looks up a field path (e.g., ["input", "name"]) in a type or shape
func (tc *TypeChecker) lookupFieldPath(baseType ast.TypeNode, fieldPath []string) (ast.TypeNode, error) {
	if len(fieldPath) == 0 {
		return baseType, nil
	}
	fieldName := ast.Ident{ID: ast.Identifier(fieldPath[0])}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldPath",
		"baseType":  baseType.Ident,
		"fieldName": fieldName.ID,
		"fieldPath": fieldPath,
		"fullPath":  fmt.Sprintf("%v", fieldPath),
	}).Debugf("=== FIELD PATH LOOKUP DEBUG ===")

	// Resolve type aliases before lookup
	resolvedType := tc.resolveTypeAliasChain(baseType)

	// *T.field — look up on T (Go data fields on pointers)
	if resolvedType.Ident == ast.TypePointer && len(resolvedType.TypeParams) > 0 {
		return tc.lookupFieldPath(resolvedType.TypeParams[0], fieldPath)
	}

	// Opaque Go struct: allow a few enmime/* field names so len()/returns typecheck; else implicit.
	if resolvedType.Ident == ast.TypeImplicit {
		if ft, ok := implicitGoFieldType(string(fieldName.ID)); ok {
			if len(fieldPath) == 1 {
				return ft, nil
			}
			return tc.lookupFieldPath(ft, fieldPath[1:])
		}
		if len(fieldPath) == 1 {
			return ast.TypeNode{Ident: ast.TypeImplicit}, nil
		}
		return tc.lookupFieldPath(ast.TypeNode{Ident: ast.TypeImplicit}, fieldPath[1:])
	}

	tc.log.WithFields(logrus.Fields{
		"function":     "lookupFieldPath",
		"baseType":     baseType.Ident,
		"resolvedType": resolvedType.Ident,
		"fieldName":    fieldName.ID,
	}).Debugf("Resolved type alias")

	// Try type definition lookup
	if def, exists := tc.Defs[resolvedType.Ident]; exists {
		tc.log.WithFields(logrus.Fields{
			"function":  "lookupFieldPath",
			"baseType":  baseType.Ident,
			"fieldName": fieldName.ID,
			"defType":   fmt.Sprintf("%T", def),
			"def":       fmt.Sprintf("%+v", def),
		}).Debugf("Found type definition")

		if typeDef, ok := def.(ast.TypeDefNode); ok {
			tc.log.WithFields(logrus.Fields{
				"function":  "lookupFieldPath",
				"baseType":  baseType.Ident,
				"fieldName": fieldName.ID,
				"exprType":  fmt.Sprintf("%T", typeDef.Expr),
				"expr":      fmt.Sprintf("%+v", typeDef.Expr),
			}).Debugf("Type definition expression")

			switch expr := typeDef.Expr.(type) {
			case ast.TypeDefAssertionExpr:
				// Handle assertion types by resolving the assertion
				if expr.Assertion != nil {
					tc.log.WithFields(logrus.Fields{
						"function":  "lookupFieldPath",
						"baseType":  baseType.Ident,
						"fieldName": fieldName.ID,
						"assertion": fmt.Sprintf("%+v", expr.Assertion),
					}).Debugf("Looking up field in assertion expression")

					// Use lookupFieldInAssertion to resolve the assertion
					return tc.lookupFieldInAssertion(resolvedType, fieldName, expr.Assertion)
				}
				return ast.TypeNode{}, fmt.Errorf("assertion has no constraints")
			case ast.TypeDefShapeExpr:
				return tc.lookupFieldPathFromPayload(baseType, fieldName, fieldPath, &expr.Shape)
			case ast.TypeDefErrorExpr:
				return tc.lookupFieldPathFromPayload(baseType, fieldName, fieldPath, &expr.Payload)
			}
		}
	}

	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldPath",
		"baseType":  baseType.Ident,
		"fieldName": fieldName.ID,
		"fieldPath": fieldPath,
		"result":    "not found",
	}).Debugf("=== END FIELD PATH LOOKUP DEBUG ===")

	return ast.TypeNode{}, fmt.Errorf("field path %v not found in type %s", fieldPath, baseType.Ident)
}

// lookupFieldPathFromPayload handles field lookup for TypeDefShapeExpr and TypeDefErrorExpr (same payload shape semantics).
func (tc *TypeChecker) lookupFieldPathFromPayload(
	baseType ast.TypeNode,
	fieldName ast.Ident,
	fieldPath []string,
	payload *ast.ShapeNode,
) (ast.TypeNode, error) {
	if payload == nil {
		return ast.TypeNode{}, fmt.Errorf("lookupFieldPathFromPayload: nil payload")
	}
	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldPath",
		"baseType":  baseType.Ident,
		"fieldName": fieldName.ID,
		"shape":     fmt.Sprintf("%+v", *payload),
		"fields":    fmt.Sprintf("%+v", payload.Fields),
	}).Debugf("Looking up field in typedef payload")

	field, exists := payload.Fields[string(fieldName.ID)]
	if !exists {
		tc.log.WithFields(logrus.Fields{
			"function":        "lookupFieldPath",
			"baseType":        baseType.Ident,
			"fieldName":       fieldName.ID,
			"availableFields": fmt.Sprintf("%+v", payload.Fields),
		}).Debugf("Field not found in typedef payload")
		return ast.TypeNode{}, fmt.Errorf("field %s not found in shape", fieldName.ID)
	}

	tc.log.WithFields(logrus.Fields{
		"function":     "lookupFieldPath",
		"baseType":     baseType.Ident,
		"fieldName":    fieldName.ID,
		"field":        fmt.Sprintf("%+v", field),
		"fieldType":    fmt.Sprintf("%T", field),
		"hasType":      field.Type != nil,
		"hasShape":     field.Shape != nil,
		"hasAssertion": field.Assertion != nil,
	}).Debugf("Found field in typedef payload")

	if field.Type != nil && len(fieldPath) == 1 {
		resolvedFieldType := tc.resolveTypeAliasChain(*field.Type)
		tc.log.WithFields(logrus.Fields{
			"function":          "lookupFieldPath",
			"baseType":          baseType.Ident,
			"fieldName":         fieldName.ID,
			"fieldType":         field.Type.Ident,
			"resolvedFieldType": resolvedFieldType.Ident,
		}).Debugf("Resolved field type alias")
		return resolvedFieldType, nil
	}
	if field.Shape != nil && len(fieldPath) > 1 {
		return tc.lookupFieldPathOnShape(field.Shape, fieldPath[1:])
	}
	if field.Shape != nil && len(fieldPath) == 1 {
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}
	if field.Type != nil && len(fieldPath) > 1 {
		return tc.lookupFieldPath(*field.Type, fieldPath[1:])
	}
	if field.Assertion != nil && len(fieldPath) == 1 {
		if field.Assertion.BaseType != nil {
			resolvedField := tc.resolveTypeAliasChain(ast.TypeNode{Ident: *field.Assertion.BaseType})
			tc.log.WithFields(logrus.Fields{
				"function":          "lookupFieldPath",
				"baseType":          baseType.Ident,
				"fieldName":         fieldName.ID,
				"assertionBaseType": *field.Assertion.BaseType,
				"resolvedType":      resolvedField.Ident,
			}).Debugf("Resolved assertion field")
			return resolvedField, nil
		}

		if len(field.Assertion.Constraints) > 0 && field.Assertion.Constraints[0].Name == ast.ValueConstraint {
			return tc.inferValueConstraintType(field.Assertion.Constraints[0], string(fieldName.ID), nil)
		}
	}
	if field.Assertion != nil && len(fieldPath) > 1 {
		if field.Assertion.BaseType != nil {
			return tc.lookupFieldPath(ast.TypeNode{Ident: *field.Assertion.BaseType}, fieldPath[1:])
		}
	}
	tc.log.WithFields(logrus.Fields{
		"function":  "lookupFieldPath",
		"baseType":  baseType.Ident,
		"fieldName": fieldName.ID,
		"field":     fmt.Sprintf("%+v", field),
	}).Debugf("Field exists but is not a type or shape")
	return ast.TypeNode{}, fmt.Errorf("field %s exists but is not a type or shape", fieldName.ID)
}

// lookupFieldPathOnShape recursively looks up a field path in a ShapeNode
func (tc *TypeChecker) lookupFieldPathOnShape(shape *ast.ShapeNode, fieldPath []string) (ast.TypeNode, error) {
	if shape == nil || len(fieldPath) == 0 {
		return ast.TypeNode{}, fmt.Errorf("invalid shape or empty path")
	}
	fieldName := fieldPath[0]
	field, exists := shape.Fields[fieldName]
	if !exists {
		return ast.TypeNode{}, fmt.Errorf("field %s not found in shape", fieldName)
	}
	if field.Type != nil && len(fieldPath) == 1 {
		// Resolve type aliases even for single-segment paths
		return tc.resolveTypeAliasChain(*field.Type), nil
	}
	if field.Shape != nil && len(fieldPath) > 1 {
		return tc.lookupFieldPathOnShape(field.Shape, fieldPath[1:])
	}
	if field.Shape != nil && len(fieldPath) == 1 {
		return ast.TypeNode{Ident: ast.TypeShape}, nil
	}
	if field.Type != nil && len(fieldPath) > 1 {
		// If the field is a type alias, resolve and continue
		return tc.lookupFieldPath(*field.Type, fieldPath[1:])
	}
	if field.Type != nil {
		return *field.Type, nil
	}
	// Handle Value constraints (like id: query.id)
	if field.Assertion != nil && len(field.Assertion.Constraints) > 0 && field.Assertion.Constraints[0].Name == ast.ValueConstraint {
		return tc.inferValueConstraintType(field.Assertion.Constraints[0], fieldName, nil)
	}
	return ast.TypeNode{}, fmt.Errorf("field %s exists but is not a type or shape", fieldName)
}

// implicitGoFieldType maps a subset of common github.com/jhillyerd/enmime Envelope fields for FFI typing.
func implicitGoFieldType(fieldName string) (ast.TypeNode, bool) {
	switch fieldName {
	case "Attachments":
		return ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{{Ident: ast.TypeImplicit}}}, true
	case "Text", "HTML":
		return ast.TypeNode{Ident: ast.TypeString}, true
	default:
		return ast.TypeNode{}, false
	}
}
