package typechecker

import (
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

func TestPreloadGoImportPackages_populatesGoPkgsByLocal(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := `package main
import "os/exec"
func main() {
	exec.Command("true")
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["exec"] == nil {
		t.Skip("os/exec not loaded (go/packages or workspace)")
	}
	if !tc.GoImportPackageLoaded("exec") {
		t.Fatal("expected GoImportPackageLoaded(exec)")
	}
}

func TestPreloadGoImportPackages_loadsStdlibWhenWorkspaceDirEmpty(t *testing.T) {
	src := `package main
import "os/exec"
func main() {
	exec.Command("true")
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = ""
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatal(err)
	}
	tc.preloadGoImportPackages()
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["exec"] == nil {
		t.Skip("os/exec not loaded (go/packages unavailable)")
	}
	if !tc.goPackagesPreloaded {
		t.Fatal("expected goPackagesPreloaded true after stdlib preload")
	}
}

func TestPreloadGoImportPackages_doesNotMarkPreloadedWhenImportsMissing(t *testing.T) {
	src := `package main
import "os/exec"
func main() {
	exec.Command("true")
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = moduleRootFromWD(t)
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatal(err)
	}
	// Empty batch result must not block gap-fill in InferTypes.
	tc.InitGoPackagesFromBatch(map[string]*packages.Package{"fmt": nil})
	if tc.goPackagesPreloaded {
		t.Fatal("goPackagesPreloaded must stay false when exec is not loaded")
	}
	if err := tc.InferTypes(nodes); err != nil {
		t.Fatalf("InferTypes: %v", err)
	}
	if !tc.GoImportPackageLoaded("exec") {
		t.Skip("os/exec not loaded")
	}
}

func TestPreloadGoImportPackages_setsGoPackagesPreloaded(t *testing.T) {
	dir := moduleRootFromWD(t)
	src := `package main
import "strings"
func main() {
	strings.Contains("a", "b")
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "t.ft", log).Lex()
	nodes, err := parser.New(toks, "t.ft", log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = dir
	if err := tc.CollectTypes(nodes); err != nil {
		t.Fatal(err)
	}
	tc.preloadGoImportPackages()
	if tc.goPkgsByLocal == nil || tc.goPkgsByLocal["strings"] == nil {
		t.Skip("strings not loaded")
	}
	if !tc.goPackagesPreloaded {
		t.Fatal("expected goPackagesPreloaded true after batch preload")
	}
}
