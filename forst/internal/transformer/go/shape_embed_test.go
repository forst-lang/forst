package transformergo

import (
	goast "go/ast"
	"testing"

	fast "forst/internal/ast"
)

func TestTransformShapeType_embeddedFieldOmitsName(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	innerType := fast.TypeIdent("Inner")
	expr, err := tr.transformShapeType(&fast.ShapeNode{
		Fields: map[string]fast.ShapeFieldNode{
			"Inner": {
				Embedded: true,
				Type:     &fast.TypeNode{Ident: innerType, TypeKind: fast.TypeKindUserDefined},
			},
		},
		FieldOrder: []string{"Inner"},
	})
	if err != nil {
		t.Fatal(err)
	}
	st := (*expr).(*goast.StructType)
	field := st.Fields.List[0]
	if len(field.Names) != 0 {
		t.Fatalf("embedded field should omit Names, got %#v", field.Names)
	}
	if field.Type == nil {
		t.Fatal("expected type on embedded field")
	}
}
