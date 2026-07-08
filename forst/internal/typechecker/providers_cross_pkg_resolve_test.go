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

	authMerged, _, err := forstpkg.ParseAndMergePackage(nil, []string{filepath.Join(root, "auth", "log.ft")})
	if err != nil {
		t.Fatal(err)
	}
	authTC := typechecker.New(nil, false)
	authTC.GoWorkspaceDir = goRoot
	if err := authTC.CheckTypes(authMerged); err != nil {
		t.Fatalf("auth: %v", err)
	}
	if _, ok := authTC.Functions["LogEvent"]; !ok {
		t.Fatalf("missing LogEvent in %v", authTC.Functions)
	}

	apiMerged, _, err := forstpkg.ParseAndMergePackage(nil, []string{filepath.Join(root, "api", "handle.ft")})
	if err != nil {
		t.Fatal(err)
	}

	view := &stubModuleView{
		importMap: map[string]string{"providers_cross_pkg_demo/auth": "auth"},
		pkgs:      map[string]*typechecker.TypeChecker{"auth": authTC},
	}
	apiTC := typechecker.New(nil, false)
	apiTC.GoWorkspaceDir = goRoot
	apiTC.SetForstPackage("api")
	apiTC.SetModuleResult(view)
	if err := apiTC.CheckTypes(apiMerged); err != nil {
		t.Fatalf("api: %v", err)
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
