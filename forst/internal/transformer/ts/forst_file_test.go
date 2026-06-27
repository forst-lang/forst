package transformerts

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestTransformForstFileFromPath_strictTypecheck_rejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(ft, []byte("not valid forst {{{"), 0644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	_, err := TransformForstFileFromPath(ft, log, TransformForstFileOptions{RelaxedTypecheck: false})
	if err == nil {
		t.Fatal("expected error for unparseable file")
	}
}

func TestTransformForstFileFromPath_relaxed_allowsTransformAfterTypecheckFailure(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "x.ft")
	// Valid parse but may fail typecheck depending on checker — use minimal package
	src := `package main

func Broken() {
	return 1
}
`
	if err := os.WriteFile(ft, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	out, err := TransformForstFileFromPath(ft, log, TransformForstFileOptions{RelaxedTypecheck: true})
	if err != nil {
		t.Fatalf("relaxed path should still return output: %v", err)
	}
	if out == nil || out.SourceFileStem != "x" {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestTransformForstFileFromPath_gatewayExample_emitsSidecarGatewayTypes(t *testing.T) {
	hello := filepath.Join("..", "..", "..", "..", "examples", "in", "gateway", "hello.ft")
	abs, err := filepath.Abs(hello)
	if err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	out, err := TransformForstFileFromPath(abs, log, TransformForstFileOptions{RelaxedTypecheck: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Functions) < 2 {
		t.Fatalf("expected gateway functions, got %#v", out.Functions)
	}
	ts := out.GenerateTypesFile()
	if !strings.Contains(ts, "import('@forst/sidecar').ForstRoutedRequest") {
		t.Fatalf("types.d.ts should map gateway.GatewayRequest to sidecar ForstRoutedRequest:\n%s", ts)
	}
	if !strings.Contains(ts, "import('@forst/sidecar').ForstRoutedResponse") {
		t.Fatalf("types.d.ts should map gateway.GatewayResponse to sidecar ForstRoutedResponse:\n%s", ts)
	}
}
