package generators

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

func TestGenerateGoCode_sortsFuncDeclsByName(t *testing.T) {
	f := &ast.File{
		Name: ast.NewIdent("p"),
		Decls: []ast.Decl{
			&ast.FuncDecl{
				Name: ast.NewIdent("Zed"),
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Body: &ast.BlockStmt{},
			},
			&ast.FuncDecl{
				Name: ast.NewIdent("Alpha"),
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Body: &ast.BlockStmt{},
			},
		},
	}

	out, err := GenerateGoCode(f)
	if err != nil {
		t.Fatal(err)
	}
	iZed := strings.Index(out, "func Zed")
	iAlpha := strings.Index(out, "func Alpha")
	if iAlpha == -1 || iZed == -1 {
		t.Fatalf("expected both funcs in output:\n%s", out)
	}
	if iAlpha >= iZed {
		t.Fatalf("expected Alpha before Zed, got:\n%s", out)
	}
}

func TestGenerateGoCode_sortsStructFieldsByName(t *testing.T) {
	f := &ast.File{
		Name: ast.NewIdent("p"),
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{
						Name: ast.NewIdent("S"),
						Type: &ast.StructType{
							Fields: &ast.FieldList{
								List: []*ast.Field{
									{Names: []*ast.Ident{ast.NewIdent("zebra")}, Type: ast.NewIdent("int")},
									{Names: []*ast.Ident{ast.NewIdent("apple")}, Type: ast.NewIdent("string")},
								},
							},
						},
					},
				},
			},
		},
	}

	out, err := GenerateGoCode(f)
	if err != nil {
		t.Fatal(err)
	}
	iApple := strings.Index(out, "apple")
	iZebra := strings.Index(out, "zebra")
	if iApple == -1 || iZebra == -1 {
		t.Fatalf("expected both fields in output:\n%s", out)
	}
	if iApple >= iZebra {
		t.Fatalf("expected apple before zebra in struct, got:\n%s", out)
	}
}

func TestGenerateGoCode_declarationOrderGroups(t *testing.T) {
	// Unsorted input: type, func, const — output order should be const, var, type, func.
	f := &ast.File{
		Name: ast.NewIdent("p"),
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{
					&ast.TypeSpec{Name: ast.NewIdent("T"), Type: ast.NewIdent("int")},
				},
			},
			&ast.FuncDecl{
				Name: ast.NewIdent("F"),
				Type: &ast.FuncType{Params: &ast.FieldList{}},
				Body: &ast.BlockStmt{},
			},
			&ast.GenDecl{
				Tok: token.CONST,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names:  []*ast.Ident{ast.NewIdent("C")},
						Values: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}},
					},
				},
			},
		},
	}

	out, err := GenerateGoCode(f)
	if err != nil {
		t.Fatal(err)
	}
	iConst := strings.Index(out, "const")
	iType := strings.Index(out, "type T")
	iFunc := strings.Index(out, "func F")
	if iConst == -1 || iType == -1 || iFunc == -1 {
		t.Fatalf("missing expected decls:\n%s", out)
	}
	if !(iConst < iType && iType < iFunc) {
		t.Fatalf("expected const then type then func, got:\n%s", out)
	}
}
