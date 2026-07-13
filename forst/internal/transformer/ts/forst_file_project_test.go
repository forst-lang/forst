package transformerts

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestGenerateTypeScriptOutputsByPackage_twoFilesSamePackage(t *testing.T) {
	dir := t.TempDir()
	authDir := filepath.Join(dir, "auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "types.ft"), []byte(`package auth

type Token = { value: String }
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "api.ft"), []byte(`package auth

func Login(t Token) {
	return { ok: true }
}
`), 0644); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(authDir, "types.ft"),
		filepath.Join(authDir, "api.ft"),
	}
	outputs, err := GenerateTypeScriptOutputsByPackage(paths, logrus.New(), nil)
	if err != nil {
		t.Fatalf("GenerateTypeScriptOutputsByPackage: %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("outputs: got %d want 1", len(outputs))
	}
	if outputs[0].PackageName != "auth" || outputs[0].SourceFileStem != "auth" {
		t.Fatalf("package output: %#v", outputs[0])
	}
	if !strings.Contains(strings.Join(outputs[0].Types, "\n"), "Token") {
		t.Fatal("expected Token type")
	}
	if len(outputs[0].Functions) != 1 || outputs[0].Functions[0].Name != "Login" {
		t.Fatalf("functions: %#v", outputs[0].Functions)
	}
}

func TestValidateDiscoveredFileStems_mismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.ft")
	if err := os.WriteFile(path, []byte("package bar\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err := ValidateDiscoveredFileStems([]string{path}, false, logrus.New())
	if err == nil || !strings.Contains(err.Error(), "must match declared package") {
		t.Fatalf("got %v", err)
	}
}

func TestStemMatchesPackage_parentDirectory(t *testing.T) {
	if !StemMatchesPackage("/proj/auth/login.ft", "auth") {
		t.Fatal("expected parent directory match")
	}
	if StemMatchesPackage("/proj/foo.ft", "bar") {
		t.Fatal("expected mismatch")
	}
}

func TestParseMergedTypecheckProject_emptyInput(t *testing.T) {
	_, _, err := ParseMergedTypecheckProject(nil, logrus.New())
	if err == nil {
		t.Fatal("expected error for empty file list")
	}
}

// Two-file package: type in one file, function using it in another — must typecheck when merged.
func TestParseMergedTypecheckProject_twoFilesCrossReferences(t *testing.T) {
	dir := t.TempDir()
	typesSrc := `package main

type R = {
	x: Int
}
`
	usesSrc := `package main

func GetX(r R): Int {
	return r.x
}
`
	if err := os.WriteFile(filepath.Join(dir, "types.ft"), []byte(typesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "uses.ft"), []byte(usesSrc), 0644); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		filepath.Join(dir, "types.ft"),
		filepath.Join(dir, "uses.ft"),
	}
	chunks, tc, err := ParseMergedTypecheckProject(paths, logrus.New())
	if err != nil {
		t.Fatalf("ParseMergedTypecheckProject: %v", err)
	}
	if tc == nil {
		t.Fatal("expected typechecker")
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks: got %d want 2", len(chunks))
	}
	if chunks[0].Stem != "types" || chunks[1].Stem != "uses" {
		t.Fatalf("stems: %#v %#v", chunks[0].Stem, chunks[1].Stem)
	}
	if len(chunks[0].Nodes) == 0 || len(chunks[1].Nodes) == 0 {
		t.Fatal("expected non-empty AST per file")
	}

	outputs, err := GenerateTypeScriptOutputsPerFile(chunks, tc, logrus.New(), nil)
	if err != nil {
		t.Fatalf("GenerateTypeScriptOutputsPerFile: %v", err)
	}
	if len(outputs) != 2 {
		t.Fatalf("TS outputs: got %d want 2", len(outputs))
	}
}

func TestParseMergedTypecheckProject_sidecarExportRejectsPublicWithProviders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.ft")
	src := `package main

type Logger = { info(msg String) }

func PublicApi() {
	use logger: Logger
}
`
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := ParseMergedTypecheckProject([]string{path}, logrus.New())
	if err == nil {
		t.Fatal("expected sidecar export error for public function with Providers")
	}
	if !strings.Contains(err.Error(), "cannot export PublicApi") {
		t.Fatalf("expected sidecar export error, got: %v", err)
	}
}

func TestGenerateTypeScriptOutputsPerFile_wrapsPerFileTransformErrors(t *testing.T) {
	tc := typechecker.New(logrus.New(), false)
	tc.Defs["Bad"] = ast.TypeDefNode{
		Ident: "Bad",
		Expr:  ast.TypeDefAssertionExpr{Assertion: nil},
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	chunks := []ForstFileChunk{{
		Path: "bad.ft",
		Stem: "bad",
		Nodes: []ast.Node{
			ast.PackageNode{Ident: ast.Ident{ID: "main"}},
		},
	}}
	_, err := GenerateTypeScriptOutputsPerFile(chunks, tc, logger, nil)
	if err == nil || !strings.Contains(err.Error(), "bad.ft:") {
		t.Fatalf("expected wrapped chunk path error, got %v", err)
	}
}

func TestParseMergedTypecheckProject_missingFileErrors(t *testing.T) {
	_, _, err := ParseMergedTypecheckProject([]string{"/definitely/missing/file.ft"}, logrus.New())
	if err == nil {
		t.Fatal("expected parse/merge error for missing file")
	}
}

func TestParseMergedTypecheckProject_typecheckFailure(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.ft")
	src := `package main

func Broken(x UnknownType) {
	return x
}
`
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := ParseMergedTypecheckProject([]string{p}, logrus.New())
	if err == nil || !strings.Contains(err.Error(), "failed to type check") {
		t.Fatalf("expected typecheck failure, got %v", err)
	}
}
