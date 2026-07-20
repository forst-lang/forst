package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/modulecheck"
)

func TestCollectInvokeFunctionsFromModuleResult_matchesFullScan(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module modtest\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bcrypt.ft"), []byte(`package bcrypt
func Hash() { return {h: "x"} }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	full, err := CollectInvokeFunctionsFromModule(nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	modResult, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{
		ModuleRoot:   dir,
		BoundaryRoot: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	fromResult := CollectInvokeFunctionsFromModuleResult(modResult)
	if len(fromResult) != len(full) {
		t.Fatalf("len mismatch: fromResult=%d full=%d", len(fromResult), len(full))
	}
	fullNames := namesFromFns(full)
	resultNames := namesFromFns(fromResult)
	if strings.Join(fullNames, ",") != strings.Join(resultNames, ",") {
		t.Fatalf("export mismatch: fromResult=%v full=%v", resultNames, fullNames)
	}
}

func namesFromFns(fns []FunctionInfo) []string {
	var names []string
	for _, fn := range fns {
		names = append(names, fn.Package+"."+fn.Name)
	}
	return names
}

func TestCollectInvokeFunctionsFromModule_crossPackage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module modtest\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bcrypt.ft"), []byte(`package bcrypt
func Hash() { return {h: "x"} }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	fns, err := CollectInvokeFunctionsFromModule(nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, fn := range fns {
		names = append(names, fn.Package+"."+fn.Name)
	}
	if !strings.Contains(strings.Join(names, ","), "bcrypt.Hash") {
		t.Fatalf("expected bcrypt.Hash, got %v", names)
	}
}

func TestCollectInvokeFunctionsFromModule_forstGomodSubdirLayout(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte("module example.com/app/forst\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	forstDir := filepath.Join(dir, "forst")
	if err := os.MkdirAll(forstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstDir, "main.ft"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstDir, "helper.ft"), []byte(`package helper

func Ping() {
  return {ok: "yes"}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	fns, err := CollectInvokeFunctionsFromModule(nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, fn := range fns {
		names = append(names, fn.Package+"."+fn.Name)
	}
	if !strings.Contains(strings.Join(names, ","), "helper.Ping") {
		t.Fatalf("expected helper.Ping, got %v", names)
	}
}

func TestCrossPackageInvokeExports_filtersCompiledPackageAndMain(t *testing.T) {
	fns := []FunctionInfo{
		{Package: "main", Name: "main", Runnable: true},
		{Package: "main", Name: "Run", Runnable: true},
		{Package: "bcrypt", Name: "Hash", Runnable: true},
		{Package: "types", Name: "Skip", Runnable: false},
		{Package: "helper", Name: "Ping", Runnable: true},
	}
	out := CrossPackageInvokeExports(fns, "main")
	if len(out) != 2 {
		t.Fatalf("len = %d want 2: %#v", len(out), out)
	}
	got := namesFromFns(out)
	want := "bcrypt.Hash,helper.Ping"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %v want %s", got, want)
	}
}
