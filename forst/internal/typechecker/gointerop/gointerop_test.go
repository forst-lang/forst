package gointerop_test

import (
	"go/types"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker/gointerop"
)

func TestTypeToForstType_mapsErrorInterface(t *testing.T) {
	t.Parallel()
	errIface := gointerop.ErrorInterfaceType()
	if errIface == nil {
		t.Fatal("expected error interface")
	}
	got, ok := gointerop.TypeToForstType(errIface)
	if !ok || got.Ident != ast.TypeError {
		t.Fatalf("want Error, got %v ok=%v", got, ok)
	}
}

func TestIdentifierExported(t *testing.T) {
	t.Parallel()
	if !gointerop.IdentifierExported("Foo") {
		t.Fatal("Foo should be exported")
	}
	if gointerop.IdentifierExported("foo") {
		t.Fatal("foo should not be exported")
	}
}

func TestCheckSignature_multiReturnMapsToTuple(t *testing.T) {
	t.Parallel()
	r0 := types.NewVar(0, nil, "", types.Typ[types.Int])
	r1 := types.NewVar(0, nil, "", types.Universe.Lookup("error").Type())
	results := types.NewTuple(r0, r1)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), results, false)

	host := stubHost{}
	diag := func(_ ast.SourceSpan, _, _, _, _ string) error {
		return nil
	}
	got, err := gointerop.CheckSignature(host, diag, gointerop.SignatureCheck{
		Sig:             sig,
		Qual:            "test.F",
		WantSingleValue: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsTupleType() {
		t.Fatalf("want Tuple, got %v", got)
	}
}

type stubHost struct{}

func (stubHost) ForstTypeForGoType(_ types.Type) (ast.TypeNode, bool) {
	return ast.TypeNode{}, false
}

func (stubHost) IsTypeCompatible(_, _ ast.TypeNode) bool {
	return false
}

func (stubHost) GoTypeForForstType(_ ast.TypeNode) types.Type {
	return nil
}

func (stubHost) InferExpressionType(_ ast.ExpressionNode) ([]ast.TypeNode, error) {
	return nil, nil
}

func (stubHost) GoTypeForExpression(_ ast.ExpressionNode) types.Type {
	return nil
}

func TestCheckParamAssignability_mapAndIotaArgSpans(t *testing.T) {
	t.Parallel()
	mapSpan := ast.SourceSpan{StartLine: 4, StartCol: 2, EndLine: 4, EndCol: 18}
	iotaSpan := ast.SourceSpan{StartLine: 5, StartCol: 3, EndLine: 5, EndCol: 7}
	entryKeySpan := ast.SourceSpan{StartLine: 8, StartCol: 1, EndLine: 8, EndCol: 4}

	cases := []struct {
		name string
		arg  ast.ExpressionNode
		want ast.SourceSpan
	}{
		{
			name: "mapLiteralUsesSpan",
			arg: ast.MapLiteralNode{
				Span: mapSpan,
				Entries: []ast.MapEntryNode{
					{Key: ast.StringLiteralNode{Value: "k", Span: entryKeySpan}, Value: ast.IntLiteralNode{Value: 1}},
				},
			},
			want: mapSpan,
		},
		{
			name: "mapLiteralFallsBackToEntry",
			arg: ast.MapLiteralNode{
				Entries: []ast.MapEntryNode{
					{Key: ast.StringLiteralNode{Value: "k", Span: entryKeySpan}, Value: ast.IntLiteralNode{Value: 1}},
				},
			},
			want: entryKeySpan,
		},
		{
			name: "iotaLiteralUsesSpan",
			arg:  ast.IotaLiteralNode{Span: iotaSpan},
			want: iotaSpan,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got ast.SourceSpan
			diag := func(sp ast.SourceSpan, _, _, _, _ string) error {
				got = sp
				return errDiag
			}
			err := gointerop.CheckParamAssignability(stubHost{}, diag, gointerop.ParamAssignability{
				Qual:    "pkg.F",
				Index:   0,
				GoParam: types.Typ[types.String],
				ArgType: []ast.TypeNode{{Ident: ast.TypeInt}},
				Call: ast.FunctionCallNode{
					Arguments: []ast.ExpressionNode{tc.arg},
				},
				ArgIdx: 0,
			})
			if err == nil {
				t.Fatal("expected diagnostic")
			}
			if got != tc.want {
				t.Fatalf("diag span %+v want %+v", got, tc.want)
			}
		})
	}
}

var errDiag = errSentinel("diag")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
