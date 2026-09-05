package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"maps"
	"strings"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferAssertionBaseMergedFields(assertion *ast.AssertionNode) (map[string]ast.ShapeFieldNode, error) {
	mergedFields := make(map[string]ast.ShapeFieldNode)
	if assertion.BaseType == nil {
		return mergedFields, nil
	}
	if td, ok := tc.typeDefForIdent(*assertion.BaseType); ok {
		if payload, ok := ast.PayloadShape(td.Expr); ok {
			maps.Copy(mergedFields, payload.Fields)
		}
		return mergedFields, nil
	}
	if !tc.isBuiltinType(*assertion.BaseType) {
		return nil, fmt.Errorf("base type %s not found", *assertion.BaseType)
	}
	return mergedFields, nil
}

func (tc *TypeChecker) processAssertionConstraint(constraint ast.ConstraintNode, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) error {
	if constraint.Name == ConstraintMatch {
		tc.mergeAssertionMatchConstraint(constraint, mergedFields)
		return nil
	}
	guardDef, exists := tc.Defs[ast.TypeIdent(constraint.Name)]
	if !exists {
		if isBuiltinAssertionConstraintName(constraint.Name) {
			return nil
		}
		return guardUndefinedError(constraint.Name, constraintSpan(constraint))
	}
	guardNode, err := typeGuardNodeFromDef(guardDef, constraint.Name)
	if err != nil {
		return err
	}
	if guardNode.Subject.GetIdent() == "" && isBuiltinAssertionConstraintName(constraint.Name) {
		return nil
	}
	return tc.mergeAssertionGuardConstraint(guardNode, constraint, mergedFields, fieldName)
}

func typeGuardNodeFromDef(guardDef ast.Node, constraintName string) (ast.TypeGuardNode, error) {
	switch g := guardDef.(type) {
	case *ast.TypeGuardNode:
		return *g, nil
	case ast.TypeGuardNode:
		return g, nil
	default:
		if isBuiltinAssertionConstraintName(constraintName) {
			return ast.TypeGuardNode{}, nil
		}
		return ast.TypeGuardNode{}, guardUndefinedError(constraintName, ast.SourceSpan{})
	}
}

func (tc *TypeChecker) finalizeAssertionInferredType(assertion *ast.AssertionNode, mergedFields map[string]ast.ShapeFieldNode) ([]ast.TypeNode, error) {
	if assertion.BaseType != nil && len(assertion.Constraints) == 0 && len(mergedFields) == 0 &&
		tc.isBuiltinType(*assertion.BaseType) {
		return []ast.TypeNode{{Ident: *assertion.BaseType}}, nil
	}
	hash, err := tc.Hasher.HashNode(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to hash assertion during inferAssertionType: %s", err)
	}
	typeIdent := hash.ToTypeIdent()
	RegisterHashBasedType(tc, typeIdent, tc.markGenericTypeParamShapeFields(mergedFields))
	if assertion.BaseType != nil && !strings.HasPrefix(string(*assertion.BaseType), "T_") {
		baseTypeIdent := *assertion.BaseType
		if tc.IsShapeCompatibleWithNamedType(ast.ShapeNode{Fields: mergedFields}, baseTypeIdent) {
			tc.log.WithFields(logrus.Fields{
				"function":  "inferAssertionType",
				"baseType":  baseTypeIdent,
				"typeIdent": typeIdent,
				"note":      "Preserving named type (compatible with full shape logic)",
			}).Debug("[PINPOINT] inferAssertionType: Preserving named type (compatible with full shape logic)")
			return []ast.TypeNode{{Ident: baseTypeIdent}}, nil
		}
		tc.log.WithFields(logrus.Fields{
			"function":  "inferAssertionType",
			"baseType":  baseTypeIdent,
			"typeIdent": typeIdent,
			"note":      "BaseType not compatible with full shape logic, using hash-based type",
		}).Debug("[PINPOINT] inferAssertionType: BaseType not compatible, using hash-based type")
	}
	return []ast.TypeNode{{Ident: typeIdent}}, nil
}

func (tc *TypeChecker) InferAssertionType(assertion *ast.AssertionNode, isFunctionParam bool, fieldName string, expectedType *ast.TypeNode) ([]ast.TypeNode, error) {
	tc.log.WithFields(logrus.Fields{
		"assertion":       assertion,
		"isFunctionParam": isFunctionParam,
		"fieldName":       fieldName,
		"function":        "inferAssertionType",
	}).Trace("Inferring type for assertion")

	mergedFields, err := tc.inferAssertionBaseMergedFields(assertion)
	if err != nil {
		return nil, err
	}

	if len(assertion.Constraints) == 1 && assertion.Constraints[0].Name == ast.ValueConstraint {
		resolvedType, err := tc.inferValueConstraintType(assertion.Constraints[0], fieldName, expectedType)
		if err != nil {
			return nil, err
		}
		return []ast.TypeNode{resolvedType}, nil
	}

	for _, constraint := range assertion.Constraints {
		tc.log.WithFields(logrus.Fields{
			"function":   "inferAssertionType",
			"constraint": constraint.Name,
			"args":       constraint.Args,
		}).Tracef("Processing constraint")
		if err := tc.processAssertionConstraint(constraint, &mergedFields, fieldName); err != nil {
			return nil, err
		}
	}

	return tc.finalizeAssertionInferredType(assertion, mergedFields)
}
