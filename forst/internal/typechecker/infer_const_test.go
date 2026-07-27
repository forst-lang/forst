package typechecker

import (
	"forst/internal/ast"
	"forst/internal/parser"
	"testing"
)

func TestInferConstGroup_iotaEnum(t *testing.T) {
	t.Parallel()
	src := `package main

const (
  A = iota
  B
  C
)

func main() {
  println(A)
}
`
	tc := checkConstSource(t, src)
	for _, name := range []ast.Identifier{"A", "B", "C"} {
		ty, ok := tc.VariableTypes[name]
		if !ok || len(ty) != 1 || ty[0].Ident != ast.TypeInt {
			t.Fatalf("%s types = %#v ok=%v", name, ty, ok)
		}
	}
}

func TestInferConstGroup_shiftIota(t *testing.T) {
	t.Parallel()
	src := `package main

const (
  FlagNone = 1 << iota
  FlagRead
  FlagWrite
)
`
	tc := checkConstSource(t, src)
	if !tc.isPackageConst("FlagNone") || !tc.isPackageConst("FlagWrite") {
		t.Fatal("expected const names registered")
	}
}

func TestInferConstGroup_stringConst(t *testing.T) {
	t.Parallel()
	src := `package main

const Greeting = "hello"
`
	tc := checkConstSource(t, src)
	ty, ok := tc.VariableTypes[ast.Identifier("Greeting")]
	if !ok || len(ty) != 1 || ty[0].Ident != ast.TypeString {
		t.Fatalf("Greeting types = %#v ok=%v", ty, ok)
	}
}

func TestCollectConstGroup_registersIotaNames(t *testing.T) {
	t.Parallel()
	src := `package main

const (
  A = iota
  B
)
`
	log := setupTestLogger(nil)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	// Collect only (no full CheckTypes): iota names must still be registered for LSP.
	for _, n := range nodes {
		if cg, ok := n.(ast.ConstGroupNode); ok {
			if err := tc.collectConstGroup(cg); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !tc.isPackageConst("A") || !tc.isPackageConst("B") {
		t.Fatal("expected iota consts registered during collect")
	}
	for _, name := range []ast.Identifier{"A", "B"} {
		ty, ok := tc.VariableTypes[name]
		if !ok || len(ty) != 1 || ty[0].Ident != ast.TypeInt {
			t.Fatalf("%s types = %#v ok=%v", name, ty, ok)
		}
	}
}

func TestInferConstGroup_rejectsIotaOutsideConst(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
  x := iota
}
`
	log := setupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err == nil {
		t.Fatal("expected error for iota outside const")
	}
}

func checkConstSource(t *testing.T, src string) *TypeChecker {
	t.Helper()
	log := setupTestLogger(nil)
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
	return tc
}
