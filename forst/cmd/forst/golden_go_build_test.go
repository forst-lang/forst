package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/codegen/layout"
	"forst/internal/compiler"
)

type companionGoBuildOpts struct {
	Label            string
	MainCode         string
	NodeRuntimeCode  string
	InvokeServerCode string
	ExtraPackages    map[string]string
	BoundaryRoot     string
}

type companionGoldenFilesGoBuildOpts struct {
	Label             string
	MainGoldenPath    string
	RuntimeGoldenPath string
	InvokeGoldenPath  string
	ExtraPackages     map[string]string
	BoundaryRoot      string
}

func verifyCompanionPackageGoBuild(t *testing.T, opts companionGoBuildOpts) {
	t.Helper()
	if err := compiler.BuildGoProgram(opts.MainCode, opts.NodeRuntimeCode, opts.InvokeServerCode, opts.ExtraPackages, opts.BoundaryRoot); err != nil {
		t.Fatalf("go build %s failed:\n%v", opts.Label, err)
	}
}

func verifyCompanionGoldenFilesGoBuild(t *testing.T, opts companionGoldenFilesGoBuildOpts) {
	t.Helper()
	mainCode, err := os.ReadFile(opts.MainGoldenPath)
	if err != nil {
		t.Fatalf("read main golden %s: %v", opts.MainGoldenPath, err)
	}
	var nodeRuntimeCode, invokeServerCode string
	if opts.RuntimeGoldenPath != "" {
		runtimeCode, err := os.ReadFile(opts.RuntimeGoldenPath)
		if err != nil {
			t.Fatalf("read runtime golden %s: %v", opts.RuntimeGoldenPath, err)
		}
		nodeRuntimeCode = string(runtimeCode)
	}
	if opts.InvokeGoldenPath != "" {
		invokeCode, err := os.ReadFile(opts.InvokeGoldenPath)
		if err != nil {
			t.Fatalf("read invoke golden %s: %v", opts.InvokeGoldenPath, err)
		}
		invokeServerCode = string(invokeCode)
	}
	extraPackages := opts.ExtraPackages
	if extraPackages == nil {
		extraPackages = readExtraPackagesFromGoldenDir(t, opts.MainGoldenPath)
	}
	verifyCompanionPackageGoBuild(t, companionGoBuildOpts{
		Label:            opts.Label,
		MainCode:         string(mainCode),
		NodeRuntimeCode:  nodeRuntimeCode,
		InvokeServerCode: invokeServerCode,
		ExtraPackages:    extraPackages,
		BoundaryRoot:     opts.BoundaryRoot,
	})
}

func readExtraPackagesFromGoldenDir(t *testing.T, mainGoldenPath string) map[string]string {
	t.Helper()
	outDir := filepath.Dir(mainGoldenPath)
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read golden output dir %s: %v", outDir, err)
	}
	extras := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkg := entry.Name()
		genPath := filepath.Join(outDir, pkg, pkg+layout.SuffixGen)
		code, err := os.ReadFile(genPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("read extra package golden %s: %v", genPath, err)
		}
		extras[pkg] = string(code)
	}
	return extras
}

func TestBuildGoProgram_nodeSeqInnerPointerMismatch(t *testing.T) {
	mainCode := "package main\n\nfunc main() {}\n"
	runtimeCode := `package main

import (
	"encoding/json"
	"forst/nodert"
)

type forstNodeSeq_float64 struct {
	inner nodert.Seq[float64]
}

func forst_node_open_seq_legacy_generators_ts_syncNumbers() (*forstNodeSeq_float64, error) {
	var seq, err = nodert.OpenSeqArgs[float64]("legacy/generators.ts", "syncNumbers", nodert.ExportKindGenerator, json.RawMessage("[3]"))
	if err != nil {
		return nil, err
	}
	return &forstNodeSeq_float64{inner: seq}, nil
}
`
	err := compiler.BuildGoProgram(mainCode, runtimeCode, "", nil, "")
	if err == nil {
		t.Fatal("expected go build to fail when inner stores *nodert.Seq as value")
	}
	if !strings.Contains(err.Error(), "cannot use seq") {
		t.Fatalf("unexpected build error:\n%v", err)
	}
}
