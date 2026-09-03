package typechecker

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
	"go/types"
)

func TestForstAssignableToGoType_samePkgShapeAliasToGoNamedStruct(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testmod.GoModContent("example.com/rivulet")), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.go"), []byte(`package config

type Config struct {
	Port string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	goload.ClearLoadCacheForTest()
	loaded, err := goload.LoadByPkgPath(dir, []string{"example.com/rivulet/config"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	pkg := loaded["example.com/rivulet/config"]
	if pkg == nil || pkg.Types == nil {
		t.Fatal("config package missing")
	}
	obj := pkg.Types.Scope().Lookup("Config")
	if obj == nil {
		t.Fatal("Config type missing")
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		t.Fatal("Config not a type name")
	}
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	tc.importPathByLocal = map[string]string{"config": "example.com/rivulet/config"}
	tc.goPkgsByLocal = map[string]*types.Package{"config": pkg.Types}
	tc.Defs[ast.TypeIdent("Config")] = ast.TypeDefNode{
		Ident: ast.TypeIdent("Config"),
		Expr:  ast.TypeDefShapeExpr{Shape: ast.ShapeNode{Fields: map[string]ast.ShapeFieldNode{}}},
	}

	forstArg := ast.TypeNode{Ident: ast.TypeIdent("Config")}
	if !tc.forstAssignableToGoType(forstArg, tn.Type()) {
		t.Fatal("expected Forst Config assignable to Go config.Config")
	}
}
