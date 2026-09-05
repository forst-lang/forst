package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) mergeAssertionGuardConstraint(guardNode ast.TypeGuardNode, constraint ast.ConstraintNode, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) error {
	tc.log.WithFields(logrus.Fields{
		"function":    "inferAssertionType",
		"subject":     guardNode.Subject.GetIdent(),
		"subjectType": guardNode.Subject.GetType().Ident,
		"parameters":  guardNode.Parameters(),
	}).Tracef("Subject parameter and additional parameters")

	argMap := tc.guardConstraintArgMap(guardNode, constraint)
	for _, param := range guardNode.Parameters() {
		if param.GetIdent() == guardNode.Subject.GetIdent() {
			continue
		}
		if err := tc.mergeGuardConstraintParameter(param, argMap, mergedFields, fieldName); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TypeChecker) guardConstraintArgMap(guardNode ast.TypeGuardNode, constraint ast.ConstraintNode) map[string]ast.Node {
	argMap := make(map[string]ast.Node)
	for i, arg := range constraint.Args {
		if i+1 < len(guardNode.Parameters()) {
			param := guardNode.Parameters()[i+1]
			argMap[param.GetIdent()] = arg
			tc.log.WithFields(logrus.Fields{
				"function": "inferAssertionType",
				"param":    param.GetIdent(),
				"arg":      fmt.Sprintf("%+v", arg),
			}).Tracef("Mapping parameter to argument")
		}
	}
	return argMap
}

func (tc *TypeChecker) mergeGuardConstraintParameter(param ast.ParamNode, argMap map[string]ast.Node, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) error {
	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"param":     param.GetIdent(),
		"paramType": param.GetType().Ident,
	}).Tracef("Processing parameter")

	if arg, ok := argMap[param.GetIdent()]; ok {
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"param":    param.GetIdent(),
			"arg":      fmt.Sprintf("%+v", arg),
		}).Tracef("Found argument for parameter")

		skipDefault, err := tc.mergeGuardConstraintArgument(param, arg, mergedFields, fieldName)
		if err != nil {
			return err
		}
		if skipDefault {
			return nil
		}
	}

	paramType := param.GetType().Ident
	(*mergedFields)[param.GetIdent()] = ast.ShapeFieldNode{
		Assertion: &ast.AssertionNode{BaseType: &paramType},
	}
	return nil
}

func (tc *TypeChecker) mergeGuardConstraintArgument(param ast.ParamNode, arg ast.Node, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) (bool, error) {
	argNode, ok := arg.(ast.ConstraintArgumentNode)
	if !ok {
		return false, nil
	}
	if argNode.Shape != nil {
		if err := tc.mergeGuardShapeLiteralArgument(param, argNode, mergedFields, fieldName); err != nil {
			return false, err
		}
		return true, nil
	}
	if argNode.Type == nil {
		return false, nil
	}
	if argNode.Type.Assertion != nil {
		skipDefault, err := tc.mergeGuardTypeAssertionArgument(param, argNode, mergedFields, fieldName)
		return skipDefault, err
	}
	return false, nil
}

func (tc *TypeChecker) mergeGuardShapeLiteralArgument(param ast.ParamNode, argNode ast.ConstraintArgumentNode, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) error {
	tc.log.WithFields(logrus.Fields{
		"function":    "inferAssertionType",
		"param":       param.GetIdent(),
		"shape":       argNode.Shape,
		"shapeFields": argNode.Shape.Fields,
	}).Debugf("Argument is a shape literal; merging its fields directly")
	if _, err := tc.inferShapeType(*argNode.Shape, nil); err != nil {
		return fmt.Errorf("failed to infer shape type: %w", err)
	}
	for k, v := range argNode.Shape.Fields {
		if err := tc.mergeGuardShapeLiteralField(k, v, argNode, mergedFields, fieldName); err != nil {
			return err
		}
	}
	tc.log.WithFields(logrus.Fields{
		"function":     "inferAssertionType",
		"param":        param.GetIdent(),
		"mergedFields": *mergedFields,
	}).Debugf("After merging shape literal fields")
	return nil
}

func (tc *TypeChecker) mergeGuardShapeLiteralField(k string, v ast.ShapeFieldNode, argNode ast.ConstraintArgumentNode, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) error {
	if v.Shape != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"shape":    fmt.Sprintf("%+v", *v.Shape),
		}).Debugf("Calling inferShapeType with shape node")
		nestedType, err := tc.inferShapeType(*v.Shape, nil)
		if err != nil {
			return fmt.Errorf("failed to infer nested shape type: %w", err)
		}
		(*mergedFields)[k] = ast.ShapeFieldNode{
			Type:  &ast.TypeNode{Ident: nestedType.Ident},
			Shape: v.Shape,
		}
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"field":    k,
			"type":     nestedType.Ident,
		}).Debugf("Added nested shape field from shape literal argument with registered shape type")
		return nil
	}
	if v.Type != nil && v.Type.Ident == ast.TypeShape && v.Shape == nil && argNode.Shape != nil {
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"shape":    fmt.Sprintf("%+v", *argNode.Shape),
		}).Debugf("Calling inferShapeType with shape node")
		nestedType, err := tc.inferShapeType(*argNode.Shape, nil)
		if err != nil {
			return fmt.Errorf("failed to infer nested shape type: %w", err)
		}
		(*mergedFields)[k] = ast.ShapeFieldNode{
			Type:  &ast.TypeNode{Ident: nestedType.Ident},
			Shape: argNode.Shape,
		}
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"field":    k,
			"type":     nestedType.Ident,
		}).Debugf("Fixed: Added nested shape field from argument-provided shape type")
		return nil
	}
	if v.Type != nil {
		(*mergedFields)[k] = v
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"field":    k,
			"type":     v.Type.Ident,
		}).Debugf("Added field from shape literal argument with existing type")
		return nil
	}
	if v.Type != nil && v.Type.Ident == ast.TypeShape && v.Type.Assertion != nil {
		inferredTypes, err := tc.InferAssertionType(v.Type.Assertion, false, fieldName, nil)
		if err != nil {
			return err
		}
		var inferredTypeIdent ast.TypeIdent
		if len(inferredTypes) > 0 {
			inferredTypeIdent = inferredTypes[0].Ident
		}
		(*mergedFields)[k] = ast.ShapeFieldNode{
			Type:  &ast.TypeNode{Ident: inferredTypeIdent},
			Shape: nil,
		}
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"field":    k,
			"type":     inferredTypeIdent,
		}).Debugf("Fixed: Added nested shape field from argument field assertion type")
		return nil
	}
	(*mergedFields)[k] = v
	tc.log.WithFields(logrus.Fields{
		"function": "inferAssertionType",
		"field":    k,
		"value":    v,
	}).Debugf("Added field from shape literal argument (preserving nested structure)")
	return nil
}

func (tc *TypeChecker) mergeGuardTypeAssertionArgument(param ast.ParamNode, argNode ast.ConstraintArgumentNode, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) (bool, error) {
	tc.log.WithFields(logrus.Fields{
		"function":  "inferAssertionType",
		"param":     param.GetIdent(),
		"type":      argNode.Type.Ident,
		"assertion": argNode.Type.Assertion,
	}).Tracef("Argument is a type")

	inferredTypes, err := tc.InferAssertionType(argNode.Type.Assertion, false, fieldName, nil)
	if err != nil {
		return false, err
	}
	var concreteType ast.TypeIdent
	if len(inferredTypes) > 0 {
		concreteType = inferredTypes[0].Ident
	}
	if def, ok := tc.Defs[concreteType].(ast.TypeDefNode); ok {
		if payload, ok := ast.PayloadShape(def.Expr); ok {
			matchShape := matchShapeFromAssertion(argNode.Type.Assertion)
			if err := tc.mergeGuardMatchedPayloadFields(*payload, matchShape, mergedFields, fieldName); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	(*mergedFields)[param.GetIdent()] = ast.ShapeFieldNode{
		Type:      &ast.TypeNode{Ident: concreteType},
		Assertion: nil,
		Shape:     nil,
	}
	tc.log.WithFields(logrus.Fields{
		"function": "inferAssertionType",
		"field":    param.GetIdent(),
		"type":     concreteType,
	}).Debugf("Added field with concrete type from TypeNode assertion (fallback)")
	return false, nil
}

func matchShapeFromAssertion(assertion *ast.AssertionNode) *ast.ShapeNode {
	for _, c := range assertion.Constraints {
		if c.Name == "Match" && len(c.Args) > 0 && c.Args[0].Shape != nil {
			return c.Args[0].Shape
		}
	}
	return nil
}

func (tc *TypeChecker) mergeGuardMatchedPayloadFields(payload ast.ShapeNode, matchShape *ast.ShapeNode, mergedFields *map[string]ast.ShapeFieldNode, fieldName string) error {
	for payloadFieldName, fieldNode := range payload.Fields {
		if fieldNode.Type == nil || fieldNode.Type.Ident != ast.TypeShape || matchShape == nil {
			continue
		}
		argField, ok := matchShape.Fields[payloadFieldName]
		if !ok {
			continue
		}
		if argField.Type != nil && argField.Type.Assertion != nil {
			inferredNestedTypes, err := tc.InferAssertionType(argField.Type.Assertion, false, fieldName, nil)
			if err != nil {
				return err
			}
			var nestedTypeIdent ast.TypeIdent
			if len(inferredNestedTypes) > 0 {
				nestedTypeIdent = inferredNestedTypes[0].Ident
			}
			(*mergedFields)[payloadFieldName] = ast.ShapeFieldNode{
				Type:  &ast.TypeNode{Ident: nestedTypeIdent},
				Shape: nil,
			}
			tc.log.WithFields(logrus.Fields{
				"function": "inferAssertionType",
				"field":    payloadFieldName,
				"type":     nestedTypeIdent,
			}).Debugf("Matched and inferred nested field from argument assertion")
			continue
		}
		(*mergedFields)[payloadFieldName] = argField
		tc.log.WithFields(logrus.Fields{
			"function": "inferAssertionType",
			"field":    payloadFieldName,
			"type":     argField.Type.Ident,
		}).Debugf("Matched and used argument field type for nested field")
	}
	return nil
}
