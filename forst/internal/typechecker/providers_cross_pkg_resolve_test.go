package typechecker_test

import (
	"path/filepath"
	"testing"

	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/typechecker"
)

func TestResolveForstSiblingCall_crossPkgExample(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "rfc", "providers", "cross_pkg")
	goRoot := goload.FindModuleRoot(root)

	alphaMerged, _, err := forstpkg.ParseAndMergePackage(nil, []string{filepath.Join(root, "alpha", "log.ft")})
	if err != nil {
		t.Fatal(err)
	}
	alphaTC := typechecker.New(nil, false)
	alphaTC.GoWorkspaceDir = goRoot
	if err := alphaTC.CheckTypes(alphaMerged); err != nil {
		t.Fatalf("alpha: %v", err)
	}
	if _, ok := alphaTC.Functions["LogExpiry"]; !ok {
		t.Fatalf("missing LogExpiry in %v", alphaTC.Functions)
	}

	betaMerged, _, err := forstpkg.ParseAndMergePackage(nil, []string{filepath.Join(root, "beta", "handle.ft")})
	if err != nil {
		t.Fatal(err)
	}

	view := &stubModuleView{
		importMap: map[string]string{"providers_cross_pkg_demo/alpha": "alpha"},
		pkgs:      map[string]*typechecker.TypeChecker{"alpha": alphaTC},
	}
	betaTC := typechecker.New(nil, false)
	betaTC.GoWorkspaceDir = goRoot
	betaTC.SetForstPackage("beta")
	betaTC.SetModuleResult(view)
	if err := betaTC.CheckTypes(betaMerged); err != nil {
		t.Fatalf("beta: %v", err)
	}
}

type stubModuleView struct {
	importMap map[string]string
	pkgs      map[string]*typechecker.TypeChecker
}

func (s *stubModuleView) ImportPathToForstPkg() map[string]string { return s.importMap }
func (s *stubModuleView) ForstPackageTypeChecker(pkg string) *typechecker.TypeChecker {
	return s.pkgs[pkg]
}
