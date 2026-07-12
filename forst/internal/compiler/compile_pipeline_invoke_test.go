package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompile_embeddedInvoke_emitsCompanion(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "in", "rfc", "embedded-invoke")
	mainPath := filepath.Join(root, "main.ft")
	if _, err := os.Stat(mainPath); err != nil {
		t.Skip("embedded-invoke example not present:", err)
	}
	c := New(Args{
		Command:     "build",
		FilePath:    mainPath,
		PackageRoot: root,
		LogLevel:    "error",
	}, nil)
	mainCode, _, invokeCode, _, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if invokeCode == "" {
		t.Fatal("expected invoke companion")
	}
	if !strings.Contains(mainCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("main should call ForstInvokeWaitForShutdown when companion emitted:\n%s", mainCode)
	}
	for _, want := range []string{
		"invokeembed.MustStartEmbedded",
		"forst_invoke_main_Echo",
		"ForstInvokeWaitForShutdown",
	} {
		if !strings.Contains(invokeCode, want) {
			t.Fatalf("missing %q in invoke companion:\n%s", want, invokeCode)
		}
	}
}

func TestCompile_embeddedInvoke_mainPackageNoExports_skipsShutdownAppend(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainSrc := `package main

func main() {
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{
		Command:     "build",
		FilePath:    filepath.Join(dir, "main.ft"),
		PackageRoot: dir,
		LogLevel:    "error",
	}, nil)
	mainCode, _, invokeCode, _, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if invokeCode != "" {
		t.Fatalf("expected no invoke companion, got:\n%s", invokeCode)
	}
	if strings.Contains(mainCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("main must not call ForstInvokeWaitForShutdown without companion:\n%s", mainCode)
	}
}

func TestCompile_embeddedInvoke_crossPackageExports_compiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module crosspkgtest\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte(`package main

func main() {
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bcrypt.ft"), []byte(`package bcrypt

type HashInput = {password: String}

func Hash(input HashInput) {
	return {hash: input.password}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{
		Command:     "build",
		FilePath:    filepath.Join(dir, "main.ft"),
		PackageRoot: dir,
		LogLevel:    "error",
	}, nil)
	mainCode, _, invokeCode, _, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if invokeCode == "" {
		t.Fatal("expected invoke companion with cross-package handlers")
	}
	if !strings.Contains(invokeCode, "forst_invoke_bcrypt_Hash") {
		t.Fatalf("missing bcrypt handler in companion:\n%s", invokeCode)
	}
	if !strings.Contains(invokeCode, "bcrypt.Hash") {
		t.Fatalf("missing qualified bcrypt.Hash call:\n%s", invokeCode)
	}
	if !strings.Contains(mainCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("main should call shutdown when companion present:\n%s", mainCode)
	}
}

func TestCompile_hostModeWithoutInvoke_appendsNodeShutdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{
  "server": {"embedded": true},
  "node": {
    "hostMode": true,
    "binary": "node",
    "args": ["scripts/host.mjs"]
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "host.ts"), []byte(`export function ping(): string { return "ok" }`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainSrc := `package main

import node host "./host"

func main() {
  ready := host.ping()
  ensure ready is Ok()
  println(ready)
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	c := New(Args{
		Command:     "build",
		FilePath:    filepath.Join(dir, "main.ft"),
		PackageRoot: dir,
		LogLevel:    "error",
	}, nil)
	mainCode, nodeRuntime, invokeCode, _, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if invokeCode != "" {
		t.Fatalf("expected no invoke companion, got:\n%s", invokeCode)
	}
	if nodeRuntime == "" {
		t.Fatal("expected node runtime companion")
	}
	if !strings.Contains(nodeRuntime, "ForstNodeWaitForShutdown") {
		t.Fatalf("node runtime missing shutdown helper:\n%s", nodeRuntime)
	}
	if !strings.Contains(mainCode, "ForstNodeWaitForShutdown") {
		t.Fatalf("main should call ForstNodeWaitForShutdown:\n%s", mainCode)
	}
	if strings.Contains(mainCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("main must not call invoke shutdown without invoke companion:\n%s", mainCode)
	}
}

func TestCompile_remixServe_embeddedAndHostMode(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "in", "rfc", "node-interop", "remix-serve")
	mainPath := filepath.Join(root, "main.ft")
	if _, err := os.Stat(mainPath); err != nil {
		t.Skip("remix-serve example not present:", err)
	}
	c := New(Args{
		Command:            "build",
		FilePath:           mainPath,
		PackageRoot:        root,
		ExportStructFields: true,
		LogLevel:           "error",
	}, nil)
	_, nodeRuntime, invokeCode, _, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if nodeRuntime == "" {
		t.Fatal("expected node runtime companion")
	}
	if invokeCode == "" {
		t.Fatal("expected invoke server companion")
	}
	for _, want := range []string{
		"forstNodeManifestJSON",
		"nodert.CallSync",
		"invokeembed.MustStartEmbedded",
		"forst_invoke_main_ListTodos",
	} {
		body := nodeRuntime + invokeCode
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in companions", want)
		}
	}
}
