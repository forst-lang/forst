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

func TestTransformForstFileFromPath_readError(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	_, err := TransformForstFileFromPath("/definitely/missing/file.ft", log, TransformForstFileOptions{})
	if err == nil || !strings.Contains(err.Error(), "failed to read file") {
		t.Fatalf("expected read-file error, got %v", err)
	}
}

func TestTransformForstFileFromPath_strictTypecheck_reportsTypeErrors(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad_types.ft")
	src := `package main

func Broken(x UnknownType) {
	return x
}
`
	if err := os.WriteFile(ft, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	_, err := TransformForstFileFromPath(ft, log, TransformForstFileOptions{RelaxedTypecheck: false})
	if err == nil || !strings.Contains(err.Error(), "failed to type check") {
		t.Fatalf("expected strict typecheck error, got %v", err)
	}
}

func TestTransformForstFileFromPath_relaxedTypecheck_continuesAfterTypeErrors(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad_relaxed.ft")
	src := `package main

func Broken(x UnknownType) {
	return x
}
`
	if err := os.WriteFile(ft, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	out, err := TransformForstFileFromPath(ft, log, TransformForstFileOptions{RelaxedTypecheck: true})
	if err != nil {
		t.Fatalf("expected relaxed mode to continue, got %v", err)
	}
	if out == nil || out.SourceFileStem != "bad_relaxed" {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestTransformForstFileFromPath_relaxedTypecheck_withNilLogger(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad_relaxed_nil_log.ft")
	src := `package main

func Broken(x UnknownType) {
	return x
}
`
	if err := os.WriteFile(ft, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := TransformForstFileFromPath(ft, nil, TransformForstFileOptions{RelaxedTypecheck: true})
	if err != nil {
		t.Fatalf("expected relaxed mode to continue with nil logger, got %v", err)
	}
	if out == nil || out.SourceFileStem != "bad_relaxed_nil_log" {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestTransformForstFileFromPath_strictTypecheck_success(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "ok.ft")
	src := `package main

func Echo(x String) {
	return x
}
`
	if err := os.WriteFile(ft, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	out, err := TransformForstFileFromPath(ft, log, TransformForstFileOptions{RelaxedTypecheck: false})
	if err != nil {
		t.Fatalf("expected strict successful transform, got %v", err)
	}
	if out == nil || out.SourceFileStem != "ok" {
		t.Fatalf("unexpected output: %#v", out)
	}
}
