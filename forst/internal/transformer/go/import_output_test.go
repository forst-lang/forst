package transformergo

import (
	goast "go/ast"
	"go/token"
	"testing"

	forstast "forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestTransformImport_and_ImportGroup(t *testing.T) {
	log := logrus.New()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	single := tr.transformImport(forstast.ImportNode{Path: `"fmt"`})
	if single.Tok.String() != "import" || len(single.Specs) != 1 {
		t.Fatalf("single import: %#v", single)
	}

	group := tr.transformImportGroup(forstast.ImportGroupNode{
		Imports: []forstast.ImportNode{
			{Path: `"fmt"`},
			{Path: `"strings"`},
		},
	})
	if len(group.Specs) != 2 {
		t.Fatalf("group specs: %d", len(group.Specs))
	}

	impA := tr.transformImport(forstast.ImportNode{Path: `"fmt"`, Alias: &forstast.Ident{ID: "f"}})
	spec := impA.Specs[0].(*goast.ImportSpec)
	if spec.Name == nil || spec.Name.Name != "f" {
		t.Fatalf("alias: %#v", spec.Name)
	}
}

func TestNameFromAlias(t *testing.T) {
	if nameFromAlias(nil) != nil {
		t.Fatal("nil alias")
	}
	got := nameFromAlias(&forstast.Ident{ID: "x"})
	if got == nil || got.Name != "x" {
		t.Fatalf("got %#v", got)
	}
}

func TestTransformerOutput_importsAndPackageName(t *testing.T) {
	var out TransformerOutput
	out.SetPackageName("")
	if out.PackageName() != "main" {
		t.Fatalf("empty package name: %q", out.PackageName())
	}
	out.SetPackageName("p")
	if out.PackageName() != "p" {
		t.Fatalf("package: %q", out.PackageName())
	}

	decl := &goast.GenDecl{
		Tok: token.IMPORT,
		Specs: []goast.Spec{
			&goast.ImportSpec{
				Name: goast.NewIdent("fmt"),
				Path: &goast.BasicLit{Kind: token.STRING, Value: `"fmt"`},
			},
		},
	}
	out.AddImport(decl)
	out.EnsureImport("fmt") // same local name — should not append another
	if len(out.imports) != 1 {
		t.Fatalf("imports len: %d", len(out.imports))
	}

	group := &goast.GenDecl{
		Tok: token.IMPORT,
		Specs: []goast.Spec{
			&goast.ImportSpec{Path: &goast.BasicLit{Kind: token.STRING, Value: `"os"`}},
		},
	}
	out.AddImportGroup(group)
	if len(out.importGroups) != 1 {
		t.Fatalf("import groups: %d", len(out.importGroups))
	}

	f, err := out.GenerateFile()
	if err != nil {
		t.Fatal(err)
	}
	if f.Name.Name != "p" {
		t.Fatalf("file package: %s", f.Name.Name)
	}
}

func TestTransformerOutput_HasType_and_duplicateAddType(t *testing.T) {
	var out TransformerOutput
	ts := &goast.GenDecl{
		Tok: token.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{Name: goast.NewIdent("Foo"), Type: goast.NewIdent("int")},
		},
	}
	out.AddType(ts)
	if !out.HasType("Foo") {
		t.Fatal("HasType Foo")
	}
	out.AddType(ts) // duplicate — skip
	if len(out.types) != 1 {
		t.Fatalf("types len: %d", len(out.types))
	}
}
