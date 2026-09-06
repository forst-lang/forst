package typechecker

import (
	"forst/internal/ast"
	"forst/internal/typechecker/gointerop"
)

func (tc *TypeChecker) checkShapeStructTags() {
	for ident, def := range tc.Defs {
		typeDef, ok := def.(ast.TypeDefNode)
		if !ok {
			continue
		}
		shape, ok := ast.PayloadShape(typeDef.Expr)
		if !ok || shape == nil {
			continue
		}
		changed := false
		for name, field := range shape.Fields {
			if field.Embedded || field.IsMethod || !gointerop.StructTagHasNonIgnoredJSON(field.Tag) {
				continue
			}
			field.GoExport = true
			shape.Fields[name] = field
			changed = true
			span := field.TagSpan
			goName := gointerop.ExportedFieldName(name)
			tc.warnf(span, "struct-tag-json-unexported",
				"json struct tag on field %q requires an exported Go field; generated code uses %q (enable -export-struct-fields to export all fields with json tags)",
				name, goName)
		}
		if !changed {
			continue
		}
		switch e := typeDef.Expr.(type) {
		case ast.TypeDefShapeExpr:
			e.Shape = *shape
			typeDef.Expr = e
		case ast.TypeDefErrorExpr:
			e.Payload = *shape
			typeDef.Expr = e
		default:
			continue
		}
		tc.setDef(ident, typeDef)
	}
}
