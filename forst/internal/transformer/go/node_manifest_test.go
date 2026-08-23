package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"forst/internal/typechecker"
)

func TestEmitNeedsBridgeRuntime_falseByDefault(t *testing.T) {
	t.Parallel()
	if EmitNeedsBridgeRuntime(typechecker.New(nil, false)) {
		t.Fatal("expected false without JS imports")
	}
	if EmitNeedsBridgeRuntime(nil) {
		t.Fatal("expected false for nil checker")
	}
}

func TestEmitNeedsBridgeRuntime_trueWhenSet(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetBridgeRuntimeInfo(typechecker.BridgeRuntimeInfo{NeedsBridgeRuntime: true})
	if !EmitNeedsBridgeRuntime(tc) {
		t.Fatal("expected true when NeedsBridgeRuntime set")
	}
}

func TestAppendNodeManifestDecl_emitsVar(t *testing.T) {
	t.Parallel()
	out := &TransformerOutput{}
	manifest := `{"version":1,"exports":[]}`
	AppendNodeManifestDecl(out, manifest)

	if !out.HasValueDecl(forstBridgeManifestVarName) {
		t.Fatal("expected manifest var decl")
	}
	file, err := out.GenerateFile()
	if err != nil {
		t.Fatalf("GenerateFile: %v", err)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("format: %v", err)
	}
	code := buf.String()
	if !strings.Contains(code, "var forstBridgeManifestJSON string") {
		t.Fatalf("missing manifest var: %s", code)
	}
	if !strings.Contains(code, `\"version\":1`) {
		t.Fatalf("missing manifest JSON: %s", code)
	}
}

func TestAppendNodeManifestDecl_skipsEmptyAndDedupes(t *testing.T) {
	t.Parallel()
	out := &TransformerOutput{}
	AppendNodeManifestDecl(out, "")
	if len(out.valueDecls) != 0 {
		t.Fatalf("expected no decl for empty manifest, got %d", len(out.valueDecls))
	}
	manifest := `{"version":1}`
	AppendNodeManifestDecl(out, manifest)
	AppendNodeManifestDecl(out, manifest)
	if len(out.valueDecls) != 1 {
		t.Fatalf("expected one decl, got %d", len(out.valueDecls))
	}
}

func TestAppendNodeManifestIfNeeded_onlyWhenNeeded(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tr := New(tc, nil)
	tr.AppendNodeManifestIfNeeded()
	if len(tr.Output.valueDecls) != 0 {
		t.Fatal("expected no manifest without node runtime")
	}

	tc.SetBridgeRuntimeInfo(typechecker.BridgeRuntimeInfo{
		NeedsBridgeRuntime: true,
		ManifestJSON:     `{"version":1,"exports":[]}`,
	})
	tr.appendNodeManifestToRuntime()
	if tr.BridgeRuntimeOutput == nil || !tr.BridgeRuntimeOutput.HasValueDecl(forstBridgeManifestVarName) {
		t.Fatal("expected manifest var in bridge runtime output when needsBridgeRuntime")
	}
}
