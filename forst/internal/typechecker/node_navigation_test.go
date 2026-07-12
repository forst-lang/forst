package typechecker

import (
	"path/filepath"
	"testing"

	"forst/internal/ast"
	"forst/internal/nodeinterop"
)

func TestNodeImportAbsPath_returnsBindingPath(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	tc.nodeImportsByLocal = map[string]nodeImportBinding{
		"payment": {AbsPath: "/proj/legacy/payment.ts"},
	}
	abs, ok := tc.NodeImportAbsPath("payment")
	if !ok || abs != "/proj/legacy/payment.ts" {
		t.Fatalf("NodeImportAbsPath = %q, %v", abs, ok)
	}
}

func TestNodeImportPathAbsPath_matchesImportPath(t *testing.T) {
	t.Parallel()
	tc := New(nil, false)
	tc.nodeImportsByLocal = map[string]nodeImportBinding{
		"payment": {
			Import:  ast.ImportNode{Path: "./legacy/payment", NodeOptIn: true},
			AbsPath: "/proj/legacy/payment.ts",
		},
	}
	abs, ok := tc.NodeImportPathAbsPath("./legacy/payment")
	if !ok || abs != "/proj/legacy/payment.ts" {
		t.Fatalf("NodeImportPathAbsPath = %q, %v", abs, ok)
	}
}

func TestNodeExportDefinitionLocation_withDefinitionSpan(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tsPath := filepath.Join(root, "legacy", "payment.ts")
	tc := New(nil, false)
	tc.NodeBoundaryRoot = root
	tc.nodeImportsByLocal = map[string]nodeImportBinding{
		"payment": {
			ModuleID: "legacy/payment.ts",
			AbsPath:  tsPath,
			Index: &nodeinterop.IndexV1{
				ModuleID: "legacy/payment.ts",
				Exports: []nodeinterop.IndexExport{{
					Name: "create",
					Kind: nodeinterop.ExportKindFunction,
					Definition: &nodeinterop.IndexSourceLocation{
						Line: 1, Column: 17, EndLine: 1, EndColumn: 23,
					},
				}},
			},
		},
	}
	abs, loc, ok := tc.NodeExportDefinitionLocation("payment", "create")
	if !ok || abs != tsPath || loc.Line != 1 || loc.Column != 17 {
		t.Fatalf("got abs=%q loc=%+v ok=%v", abs, loc, ok)
	}
}

func TestNodeExportDefinitionLocation_crossFileDefinition(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tc := New(nil, false)
	tc.NodeBoundaryRoot = root
	tc.nodeImportsByLocal = map[string]nodeImportBinding{
		"barrel": {
			ModuleID: "re-export-barrel.ts",
			AbsPath:  filepath.Join(root, "re-export-barrel.ts"),
			Index: &nodeinterop.IndexV1{
				ModuleID: "re-export-barrel.ts",
				Exports: []nodeinterop.IndexExport{{
					Name: "greet",
					Kind: nodeinterop.ExportKindFunction,
					Definition: &nodeinterop.IndexSourceLocation{
						File: "re-export-source.ts", Line: 1, Column: 17,
					},
				}},
			},
		},
	}
	want := filepath.Join(root, "re-export-source.ts")
	abs, _, ok := tc.NodeExportDefinitionLocation("barrel", "greet")
	if !ok || abs != want {
		t.Fatalf("abs = %q want %q ok=%v", abs, want, ok)
	}
}

func TestNodeExportDefinitionLocation_fallbackWithoutDefinition(t *testing.T) {
	t.Parallel()
	tsPath := "/proj/legacy/payment.ts"
	tc := New(nil, false)
	tc.nodeImportsByLocal = map[string]nodeImportBinding{
		"payment": {
			ModuleID: "legacy/payment.ts",
			AbsPath:  tsPath,
			Index: &nodeinterop.IndexV1{
				ModuleID: "legacy/payment.ts",
				Exports:  []nodeinterop.IndexExport{{Name: "create", Kind: nodeinterop.ExportKindFunction}},
			},
		},
	}
	abs, loc, ok := tc.NodeExportDefinitionLocation("payment", "create")
	if !ok || abs != tsPath || loc.IsSet() {
		t.Fatalf("abs=%q loc=%+v ok=%v", abs, loc, ok)
	}
}
