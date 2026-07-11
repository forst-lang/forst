package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"forst/internal/typechecker"
)

func TestEmitNeedsNodeRuntime_falseByDefault(t *testing.T) {
	t.Parallel()
	if EmitNeedsNodeRuntime(typechecker.New(nil, false)) {
		t.Fatal("expected false without node imports")
	}
	if EmitNeedsNodeRuntime(nil) {
		t.Fatal("expected false for nil checker")
	}
}

func TestEmitNeedsNodeRuntime_trueWhenSet(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(nil, false)
	tc.SetNodeRuntimeInfo(typechecker.NodeRuntimeInfo{NeedsNodeRuntime: true})
	if !EmitNeedsNodeRuntime(tc) {
		t.Fatal("expected true when NeedsNodeRuntime set")
	}
}

func TestAppendNodeManifestDecl_emitsVar(t *testing.T) {
	t.Parallel()
	out := &TransformerOutput{}
	manifest := `{"version":1,"exports":[]}`
	AppendNodeManifestDecl(out, manifest)

	if !out.HasValueDecl(forstNodeManifestVarName) {
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
	if !strings.Contains(code, "var forstNodeManifestJSON string") {
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

	tc.SetNodeRuntimeInfo(typechecker.NodeRuntimeInfo{
		NeedsNodeRuntime: true,
		ManifestJSON:     `{"version":1,"exports":[]}`,
	})
	tr.appendNodeManifestToRuntime()
	if tr.NodeRuntimeOutput == nil || !tr.NodeRuntimeOutput.HasValueDecl(forstNodeManifestVarName) {
		t.Fatal("expected manifest var in node runtime output when needsNodeRuntime")
	}
}
