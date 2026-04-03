package generators

import (
	goast "go/ast"
	"go/token"
	"testing"
)

func TestGetDeclName_all_branches(t *testing.T) {
	path := &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`}
	alias := &goast.Ident{Name: "f"}

	tests := []struct {
		name string
		decl goast.Decl
		want string
	}{
		{
			name: "FuncDecl",
			decl: &goast.FuncDecl{Name: goast.NewIdent("Foo")},
			want: "Foo",
		},
		{
			name: "GenDecl_type",
			decl: &goast.GenDecl{
				Tok: token.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{Name: goast.NewIdent("T")},
				},
			},
			want: "T",
		},
		{
			name: "GenDecl_value_first_name",
			decl: &goast.GenDecl{
				Tok: token.VAR,
				Specs: []goast.Spec{
					&goast.ValueSpec{Names: []*goast.Ident{goast.NewIdent("a"), goast.NewIdent("b")}},
				},
			},
			want: "a",
		},
		{
			name: "GenDecl_value_no_names",
			decl: &goast.GenDecl{
				Tok: token.CONST,
				Specs: []goast.Spec{
					&goast.ValueSpec{Names: []*goast.Ident{}},
				},
			},
			want: "",
		},
		{
			name: "GenDecl_import_alias",
			decl: &goast.GenDecl{
				Tok: token.IMPORT,
				Specs: []goast.Spec{
					&goast.ImportSpec{Name: alias, Path: path},
				},
			},
			want: "f",
		},
		{
			name: "GenDecl_import_path_only",
			decl: &goast.GenDecl{
				Tok: token.IMPORT,
				Specs: []goast.Spec{
					&goast.ImportSpec{Path: path},
				},
			},
			want: `"fmt"`,
		},
		{
			name: "GenDecl_empty_specs",
			decl: &goast.GenDecl{Tok: token.VAR, Specs: []goast.Spec{}},
			want: "",
		},
		{
			name: "unsupported_decl",
			decl: &goast.BadDecl{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDeclName(tt.decl); got != tt.want {
				t.Fatalf("getDeclName() = %q, want %q", got, tt.want)
			}
		})
	}
}
