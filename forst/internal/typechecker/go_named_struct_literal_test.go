package typechecker

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

func TestSamePackageGo_structLiteralAssignable(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	testmod.WriteGoMod(t, root, "example.com/baglit")
	pkgDir := filepath.Join(root, "app")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "helpers.go"), []byte(`package main

type Bag struct {
	Name string
}

func BagName(b Bag) string { return b.Name }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package main

func main() {
	b := Bag{Name: "x"}
	println(BagName(b))
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "app/main.ft", log).Lex()
	nodes, err := parser.New(toks, "app/main.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = root
	tc.SetSamePackageGoImportPath("example.com/baglit/app")
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("expected Bag literal assignable to Go Bag: %v", err)
	}
}

func TestSamePackageGo_forstFuncTakesGoNamedType(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	testmod.WriteGoMod(t, root, "example.com/bagsig")
	pkgDir := filepath.Join(root, "app")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "helpers.go"), []byte(`package main

type Bag struct {
	Name string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package main

func ForstReturnsBag(): Bag {
	return Bag{Name: "x"}
}

func ForstTakesBag(b Bag): String {
	return b.Name
}

func main() {
	println(ForstTakesBag(ForstReturnsBag()))
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "app/main.ft", log).Lex()
	nodes, err := parser.New(toks, "app/main.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = root
	tc.SetSamePackageGoImportPath("example.com/bagsig/app")
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("expected Forst funcs with Go Bag in signature: %v", err)
	}
}
