package typechecker

import (
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
)

func TestResolveForstSiblingTypeDef_returnsAlphaTypedef(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	goRoot := goload.FindModuleRoot(root)

	alphaMerged, _, err := forstpkg.ParseAndMergePackage(nil, []string{filepath.Join(root, "alpha", "log.ft")})
	if err != nil {
		t.Fatal(err)
	}
	alphaTC := New(nil, false)
	alphaTC.GoWorkspaceDir = goRoot
	if err := alphaTC.CollectTypes(alphaMerged); err != nil {
		t.Fatalf("alpha collect: %v", err)
	}

	view := &stubSiblingModuleView{
		importMap: map[string]string{"providers_cross_pkg_demo/alpha": "alpha"},
		pkgs:      map[string]*TypeChecker{"alpha": alphaTC},
	}
	betaTC := New(nil, false)
	betaTC.GoWorkspaceDir = goRoot
	betaTC.SetForstPackage("beta")
	betaTC.SetModuleResult(view)
	betaTC.imports = []ast.ImportNode{{Path: `"providers_cross_pkg_demo/alpha"`}}
	betaTC.importPathByLocal = map[string]string{"alpha": "providers_cross_pkg_demo/alpha"}

	td, ok := betaTC.resolveForstSiblingTypeDef(ast.TypeIdent("alpha.Logger"))
	if !ok {
		t.Fatal("expected alpha.Logger typedef")
	}
	if td.Ident != "Logger" {
		t.Fatalf("want Logger typedef, got %q", td.Ident)
	}
	td2, ok2 := betaTC.resolveForstSiblingTypeDef(ast.TypeIdent("alpha.Logger"))
	if !ok2 || td2.Ident != td.Ident {
		t.Fatalf("cache miss on second lookup: ok=%v ident=%q", ok2, td2.Ident)
	}
}

func TestIsTypeCompatible_siblingShapeMatchesLocalHashType(t *testing.T) {
	configShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"Version": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	alphaTC := New(nil, false)
	alphaTC.SetForstPackage("alpha")
	if err := alphaTC.CollectTypes([]ast.Node{ast.TypeDefNode{
		Ident: "Config",
		Expr:  ast.TypeDefShapeExpr{Shape: configShape},
	}}); err != nil {
		t.Fatal(err)
	}

	betaTC := New(nil, false)
	betaTC.SetForstPackage("beta")
	betaTC.SetModuleResult(&stubSiblingModuleView{
		importMap: map[string]string{"demo/alpha": "alpha"},
		pkgs:      map[string]*TypeChecker{"alpha": alphaTC},
	})
	betaTC.importPathByLocal = map[string]string{"alpha": "demo/alpha"}

	localShape := ast.TypeNode{Ident: "T_local123"}
	betaTC.Defs[localShape.Ident] = ast.TypeDefNode{
		Ident: localShape.Ident,
		Expr:  ast.TypeDefShapeExpr{Shape: configShape},
	}
	expected := ast.TypeNode{Ident: ast.TypeIdent("alpha.Config")}
	if !betaTC.IsTypeCompatible(localShape, expected) {
		t.Fatal("expected local hash shape compatible with alpha.Config")
	}
}

type stubSiblingModuleView struct {
	importMap map[string]string
	pkgs      map[string]*TypeChecker
}

func (s *stubSiblingModuleView) ImportPathToForstPkg() map[string]string { return s.importMap }
func (s *stubSiblingModuleView) ForstPackageTypeChecker(pkg string) *TypeChecker {
	return s.pkgs[pkg]
}
