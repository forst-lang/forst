package transformergo

import (
	"fmt"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

func (t *Transformer) markInlineGenericShapeParam(fn ast.Identifier, paramIndex int) {
	if t.inlineGenericShapeParams == nil {
		t.inlineGenericShapeParams = make(map[ast.Identifier]map[int]struct{})
	}
	if t.inlineGenericShapeParams[fn] == nil {
		t.inlineGenericShapeParams[fn] = make(map[int]struct{})
	}
	t.inlineGenericShapeParams[fn][paramIndex] = struct{}{}
}

func (t *Transformer) usesInlineGenericShapeParam(fn ast.Identifier, paramIndex int) bool {
	if t.inlineGenericShapeParams == nil {
		return false
	}
	_, ok := t.inlineGenericShapeParams[fn][paramIndex]
	return ok
}

func (t *Transformer) tryInlineGenericShapeParamType(fnIdent ast.Identifier, paramType ast.TypeNode) (goast.Expr, bool, error) {
	sig, ok := t.TypeChecker.Functions[fnIdent]
	if !ok || len(sig.TypeParams) == 0 {
		return nil, false, nil
	}
	shapeFields, ok := t.TypeChecker.ShapeFieldsFromParamType(paramType)
	if !ok {
		return nil, false, nil
	}
	usesTypeParam := false
	for _, sf := range shapeFields {
		tn, ok := typechecker.ShapeFieldTypeNode(sf)
		if !ok {
			continue
		}
		if tn.IsTypeParam() {
			usesTypeParam = true
			break
		}
		if sig.TypeParamNames != nil {
			if _, ok := sig.TypeParamNames[tn.Ident]; ok {
				usesTypeParam = true
				break
			}
		}
	}
	if !usesTypeParam {
		return nil, false, nil
	}
	goFields := make([]*goast.Field, 0, len(shapeFields))
	for name, sf := range shapeFields {
		tn, ok := typechecker.ShapeFieldTypeNode(sf)
		if !ok {
			continue
		}
		gt, err := t.transformType(tn)
		if err != nil {
			return nil, false, fmt.Errorf("inline generic shape field %s: %w", name, err)
		}
		goFields = append(goFields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(name)},
			Type:  gt,
		})
	}
	return &goast.StructType{Fields: &goast.FieldList{List: goFields}}, true, nil
}

func (t *Transformer) inlineStructTypeFromShapeLiteral(shape *ast.ShapeNode) (goast.Expr, error) {
	inferred, err := t.TypeChecker.LookupInferredType(shape, true)
	if err != nil || len(inferred) == 0 {
		return nil, fmt.Errorf("inline struct from shape literal: %w", err)
	}
	def, ok := t.TypeChecker.Defs[inferred[0].Ident].(ast.TypeDefNode)
	if !ok {
		return nil, fmt.Errorf("inline struct: no typedef for %s", inferred[0].Ident)
	}
	payload, ok := ast.PayloadShape(def.Expr)
	if !ok {
		return nil, fmt.Errorf("inline struct: not a shape typedef for %s", inferred[0].Ident)
	}
	goFields := make([]*goast.Field, 0, len(payload.Fields))
	for name, sf := range payload.Fields {
		tn, ok := typechecker.ShapeFieldTypeNode(sf)
		if !ok {
			return nil, fmt.Errorf("inline struct field %s: no type", name)
		}
		gt, err := t.transformType(tn)
		if err != nil {
			return nil, err
		}
		goFields = append(goFields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(name)},
		 Type:  gt,
		})
	}
	return &goast.StructType{Fields: &goast.FieldList{List: goFields}}, nil
}

func (t *Transformer) shapeTypeDefUsesGenericTypeParams(typeDef ast.TypeDefNode) bool {
	payload, ok := ast.PayloadShape(typeDef.Expr)
	if !ok {
		return false
	}
	for _, sf := range payload.Fields {
		tn, ok := typechecker.ShapeFieldTypeNode(sf)
		if !ok {
			continue
		}
		if tn.IsTypeParam() || t.TypeChecker.IsDeclaredGenericTypeParam(tn.Ident) {
			return true
		}
	}
	return false
}
