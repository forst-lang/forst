package typechecker

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"

	"github.com/sirupsen/logrus"
)

func TestPartitionTopLevelForCollect_typesBeforeFunctions(t *testing.T) {
	fn := ast.FunctionNode{}
	td := ast.TypeDefNode{Ident: "Z"}
	nodes := []ast.Node{fn, td}
	got := partitionTopLevelForCollect(nodes)
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if _, ok := got[0].(ast.TypeDefNode); !ok {
		t.Fatalf("want type def first, got %T", got[0])
	}
	if _, ok := got[1].(ast.FunctionNode); !ok {
		t.Fatalf("want function second, got %T", got[1])
	}
}

func TestCheckTypes_mergeOrder_typeDefinitionsMayFollowCallersInSource(t *testing.T) {
	dir := t.TempDir()
	typesPath := filepath.Join(dir, "types.ft")
	enginePath := filepath.Join(dir, "engine.ft")
	if err := os.WriteFile(typesPath, []byte(`package main

type R = {
	x: Int
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(enginePath, []byte(`package main

func GetX(r R): Int {
	return r.x
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(nil)
	// Merge with engine (uses R) before types (defines R) — must still typecheck.
	merged, _, err := forstpkg.ParseAndMergePackage(log, []string{enginePath, typesPath})
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = goload.FindModuleRoot(enginePath)
	if err := tc.CheckTypes(merged); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}
