package transformer_go

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"

	log "github.com/sirupsen/logrus"
)

func (t *Transformer) transformShapeFieldType(field ast.ShapeFieldNode) (*goast.Expr, error) {
	if field.Assertion != nil {
		log.Trace(fmt.Sprintf("transformShapeFieldType, assertion: %s", *field.Assertion))
		expr, err := t.transformAssertionType(field.Assertion)
		if err != nil {
			err = fmt.Errorf("failed to transform assertion type during transformation: %w", err)
			log.WithError(err).Error("transforming assertion type failed")
			return nil, err
		}
		return expr, nil
	}
	if field.Shape != nil {
		log.Trace(fmt.Sprintf("transformShapeFieldType, shape: %s", *field.Shape))
		lookupType, err := t.TypeChecker.LookupInferredType(field.Shape, true)
		if err != nil {
			err = fmt.Errorf("failed to lookup type during transformation: %w (key: %s)", err, t.TypeChecker.Hasher.HashNode(field.Shape).ToTypeIdent())
			log.WithError(err).Error("transforming type failed")
			return nil, err
		}
		log.Trace(fmt.Sprintf("transformShapeFieldType, lookupType: %s", lookupType[0]))
		shapeType := lookupType[0]
		result := transformTypeIdent(shapeType.Ident)
		var expr goast.Expr = result
		return &expr, nil
	}
	return nil, fmt.Errorf("shape field has neither assertion nor shape: %T", field)
}

func (t *Transformer) transformShapeType(shape *ast.ShapeNode) (*goast.Expr, error) {
	fields := []*goast.Field{}
	for name, field := range shape.Fields {
		fieldType, err := t.transformShapeFieldType(field)
		if err != nil {
			err = fmt.Errorf("failed to transform shape field type during transformation: %w", err)
			log.WithError(err).Error("transforming shape field type failed")
			return nil, err
		}
		fields = append(fields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(name)},
			Type:  *fieldType,
		})
	}
	result := goast.StructType{
		Fields: &goast.FieldList{
			List: fields,
		},
	}
	var expr goast.Expr = &result
	return &expr, nil
}
