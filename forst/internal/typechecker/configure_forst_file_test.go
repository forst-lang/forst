package typechecker

import (
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func TestConfigureForForstFile_setsSamePackageGoImportPath(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	root, importPath := testutil.WriteMixedGoForstModule(t, "memos")
	src := `package memos

func main() {
  x := Add(1, 2)
  println(x)
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "memos/main.ft", log).Lex()
	nodes, err := parser.New(toks, "memos/main.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.ConfigureForForstFile(root, filepath.Join(root, "memos"), nodes)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	if !tc.SamePackageGoLoaded() {
		t.Fatalf("expected same-package Go loaded for %q", importPath)
	}
}

func TestConfigureForForstFile_emptyModuleRootSkipsImportPath(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	tc.ConfigureForForstFile("", "/tmp", []ast.Node{})
	if tc.SamePackageGoLoaded() {
		t.Fatal("expected same-package Go not loaded without module root")
	}
}
