package compiler

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestCompileFile_typecheckErrorReturnsNilCode(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "bad.ft")
	if err := os.WriteFile(filePath, []byte(`package main

func Broken(): String {
	return 1
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	compiler := New(Args{
		Command:  "build",
		FilePath: filePath,
		LogLevel: "error",
	}, silentCompilerTestLogger())

	code, err := compiler.CompileFile()
	if err == nil {
		t.Fatal("expected typecheck error")
	}
	if code != nil {
		t.Fatalf("expected nil code when typecheck fails, got %q", *code)
	}
}

func TestReportPhase_logsOnlyWhenEnabled(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logBuffer)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})

	compilerWithPhaseReporting := New(Args{ReportPhases: true}, logger)
	compilerWithPhaseReporting.reportPhase("phase-enabled")
	if !strings.Contains(logBuffer.String(), "phase-enabled") {
		t.Fatalf("expected phase log when enabled, got %q", logBuffer.String())
	}

	logBuffer.Reset()
	compilerWithoutPhaseReporting := New(Args{ReportPhases: false}, logger)
	compilerWithoutPhaseReporting.reportPhase("phase-disabled")
	if strings.Contains(logBuffer.String(), "phase-disabled") {
		t.Fatalf("did not expect phase log when disabled, got %q", logBuffer.String())
	}
}

func TestLoadInputNodesForCompile_withPackageRootLogsMergedPhase(t *testing.T) {
	dir := t.TempDir()
	entryPath := filepath.Join(dir, "entry.ft")
	if err := os.WriteFile(entryPath, []byte(`package demo

func Entry(): String {
	return "ok"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "peer.ft"), []byte(`package demo

func Peer(): String {
	return "peer"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var logBuffer bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logBuffer)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})

	compiler := New(Args{
		Command:      "build",
		FilePath:     entryPath,
		PackageRoot:  dir,
		LogLevel:     "error",
		ReportPhases: true,
	}, logger)

	nodes, err := compiler.loadInputNodesForCompile()
	if err != nil {
		t.Fatalf("loadInputNodesForCompile: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected merged nodes")
	}
	if !strings.Contains(logBuffer.String(), "Loading merged package (same-package .ft files under -root)...") {
		t.Fatalf("expected merged-package phase log, got %q", logBuffer.String())
	}
}
