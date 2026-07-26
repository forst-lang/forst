package transformergo

import (
	goast "go/ast"
	"go/token"
	"testing"

	fast "forst/internal/ast"
)

func TestTransformConstGroup_single(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	decl, err := tr.transformConstGroup(fast.ConstGroupNode{
		Specs: []fast.ConstSpec{{
			Name:  fast.Ident{ID: "Pi"},
			Value: fast.IntLiteralNode{Value: 3},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decl.Tok != token.CONST {
		t.Fatalf("tok = %v", decl.Tok)
	}
	spec := decl.Specs[0].(*goast.ValueSpec)
	if spec.Names[0].Name != "Pi" {
		t.Fatalf("name = %q", spec.Names[0].Name)
	}
}

func TestTransformConstGroup_preservesIota(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	decl, err := tr.transformConstGroup(fast.ConstGroupNode{
		Specs: []fast.ConstSpec{
			{Name: fast.Ident{ID: "A"}, Value: fast.IotaLiteralNode{}},
			{Name: fast.Ident{ID: "B"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(decl.Specs) != 2 {
		t.Fatalf("specs = %d", len(decl.Specs))
	}
	first := decl.Specs[0].(*goast.ValueSpec)
	if ident, ok := first.Values[0].(*goast.Ident); !ok || ident.Name != "iota" {
		t.Fatalf("first value = %#v", first.Values[0])
	}
	second := decl.Specs[1].(*goast.ValueSpec)
	if len(second.Values) != 0 {
		t.Fatalf("repeated spec should omit value, got %#v", second.Values)
	}
}

func TestTransformConstGroup_shiftIota(t *testing.T) {
	tr := setupTransformer(setupTypeChecker(setupTestLogger(nil)), setupTestLogger(nil))
	decl, err := tr.transformConstGroup(fast.ConstGroupNode{
		Specs: []fast.ConstSpec{{
			Name: fast.Ident{ID: "FlagNone"},
			Value: fast.BinaryExpressionNode{
				Left:     fast.IntLiteralNode{Value: 1},
				Operator: fast.TokenLShift,
				Right:    fast.IotaLiteralNode{},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec := decl.Specs[0].(*goast.ValueSpec)
	bin, ok := spec.Values[0].(*goast.BinaryExpr)
	if !ok || bin.Op != token.SHL {
		t.Fatalf("value = %#v", spec.Values[0])
	}
}
