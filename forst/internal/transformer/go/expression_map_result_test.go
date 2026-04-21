package transformergo

import (
	"go/format"
	"go/token"
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestGoZeroValueGoAST_table(t *testing.T) {
	log := setupTestLogger(nil)
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	cases := []struct {
		name string
		vt   ast.TypeNode
		sub  string
	}{
		{"int", ast.NewBuiltinType(ast.TypeInt), "0"},
		{"string", ast.NewBuiltinType(ast.TypeString), `""`},
		{"bool", ast.NewBuiltinType(ast.TypeBool), "false"},
		{"float", ast.NewBuiltinType(ast.TypeFloat), "0"},
		{
			"array_int",
			ast.TypeNode{Ident: ast.TypeArray, TypeKind: ast.TypeKindBuiltin, TypeParams: []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)}},
			"",
		},
		{
			"map_string_int",
			ast.NewMapType(ast.NewBuiltinType(ast.TypeString), ast.NewBuiltinType(ast.TypeInt)),
			"",
		},
		{"pointer_int", ast.NewPointerType(ast.NewBuiltinType(ast.TypeInt)), "nil"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := tr.goZeroValueGoAST(tt.vt)
			if err != nil {
				t.Fatal(err)
			}
			var buf strings.Builder
			if err := format.Node(&buf, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			s := buf.String()
			if tt.sub != "" && !strings.Contains(s, tt.sub) {
				t.Fatalf("formatted %q should contain %q", s, tt.sub)
			}
		})
	}
}
