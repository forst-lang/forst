package main

import (
	"os"
	"strings"
	"testing"

	"forst/internal/compiler"
)

func verifyCompanionPackageGoBuild(t *testing.T, label, mainCode, nodeRuntimeCode, invokeServerCode string) {
	t.Helper()
	if err := compiler.BuildGoProgram(mainCode, nodeRuntimeCode, invokeServerCode); err != nil {
		t.Fatalf("go build %s failed:\n%v", label, err)
	}
}

func verifyCompanionGoldenFilesGoBuild(t *testing.T, label, mainGoldenPath, runtimeGoldenPath, invokeGoldenPath string) {
	t.Helper()
	mainCode, err := os.ReadFile(mainGoldenPath)
	if err != nil {
		t.Fatalf("read main golden %s: %v", mainGoldenPath, err)
	}
	var nodeRuntimeCode, invokeServerCode string
	if runtimeGoldenPath != "" {
		runtimeCode, err := os.ReadFile(runtimeGoldenPath)
		if err != nil {
			t.Fatalf("read runtime golden %s: %v", runtimeGoldenPath, err)
		}
		nodeRuntimeCode = string(runtimeCode)
	}
	if invokeGoldenPath != "" {
		invokeCode, err := os.ReadFile(invokeGoldenPath)
		if err != nil {
			t.Fatalf("read invoke golden %s: %v", invokeGoldenPath, err)
		}
		invokeServerCode = string(invokeCode)
	}
	verifyCompanionPackageGoBuild(t, label, string(mainCode), nodeRuntimeCode, invokeServerCode)
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
	err := compiler.BuildGoProgram(mainCode, runtimeCode, "")
	if err == nil {
		t.Fatal("expected go build to fail when inner stores *nodert.Seq as value")
	}
	if !strings.Contains(err.Error(), "cannot use seq") {
		t.Fatalf("unexpected build error:\n%v", err)
	}
}
