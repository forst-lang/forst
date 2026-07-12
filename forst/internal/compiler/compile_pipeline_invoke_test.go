package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/discovery"
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

func TestCompile_embeddedInvoke_crossPackage_forstGomodHostMode(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	forstDir := filepath.Join(dir, "forst")
	scriptsDir := filepath.Join(dir, "scripts")
	for _, d := range []string{forstGomod, forstDir, scriptsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	forstModule := forstCompilerModuleRoot(t)
	goMod := "module example.com/app/forst\n\ngo 1.26.0\n\nreplace forst => " + forstModule + "\n"
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	ftconfig := `{
  "server": {"embedded": true, "port": "6321"},
  "node": {
    "enabled": true,
    "hostMode": true,
    "binary": "node",
    "args": ["scripts/host.mjs"]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(ftconfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "host.mjs"), []byte("// host shim stub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstDir, "host.ts"), []byte(`export function hostPing(): string { return "ready" }`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainSrc := `package main

import node host "./host"

func main() {
	ready := host.hostPing()
	ensure ready is Ok()
	println("forst:app ready: " + ready)
}
`
	if err := os.WriteFile(filepath.Join(forstDir, "main.ft"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	bcryptSrc := `package bcrypt

type ComparePasswordRequest = {
	plainPassword: String,
	passwordHash: String
}

type ComparePasswordResponse = {
	valid: Bool
}

func ComparePassword(input ComparePasswordRequest) {
	return { valid: true }
}
`
	if err := os.WriteFile(filepath.Join(forstDir, "bcrypt.ft"), []byte(bcryptSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	moduleFns, err := discovery.CollectInvokeFunctionsFromModule(nil, dir)
	if err != nil {
		t.Fatalf("discover module exports: %v", err)
	}
	var discovered []string
	for _, fn := range moduleFns {
		discovered = append(discovered, fn.Package+"."+fn.Name)
	}
	if !strings.Contains(strings.Join(discovered, ","), "bcrypt.ComparePassword") {
		t.Fatalf("expected bcrypt.ComparePassword in discovery, got %v", discovered)
	}
	c := New(Args{
		Command:            "build",
		FilePath:           filepath.Join(forstDir, "main.ft"),
		PackageRoot:        dir,
		ExportStructFields: true,
		LogLevel:           "error",
	}, nil)
	mainCode, nodeRuntime, invokeCode, extraPkgs, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if invokeCode == "" {
		t.Fatal("expected invoke companion with cross-package handlers")
	}
	if nodeRuntime == "" {
		t.Fatal("expected node runtime companion for host mode")
	}
	bcryptPkg, ok := extraPkgs["bcrypt"]
	if !ok || bcryptPkg == "" {
		t.Fatalf("expected extra bcrypt package, got %v", extraPkgs)
	}
	for _, want := range []string{
		"forst_invoke_bcrypt_ComparePassword",
		"bcrypt.ComparePassword",
		"ForstInvokeWaitForShutdown",
		"invokeembed.MustStartEmbedded",
	} {
		if !strings.Contains(invokeCode, want) {
			t.Fatalf("missing %q in invoke companion:\n%s", want, invokeCode)
		}
	}
	if !strings.Contains(mainCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("main should call shutdown when companion present:\n%s", mainCode)
	}
	if !strings.Contains(nodeRuntime, "forstNodeManifestJSON") {
		t.Fatalf("node runtime missing manifest:\n%s", nodeRuntime)
	}
}

func TestCompile_embeddedInvoke_crossPackage_forstGomodGoFFI(t *testing.T) {
	dir := t.TempDir()
	forstGomod := filepath.Join(dir, ".forst-gomod")
	forstDir := filepath.Join(dir, "forst")
	scriptsDir := filepath.Join(dir, "scripts")
	for _, d := range []string{forstGomod, forstDir, scriptsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	forstModule := forstCompilerModuleRoot(t)
	goMod := "module example.com/app/forst\n\ngo 1.26.0\n\nrequire (\n\tforst v0.0.0\n\tgolang.org/x/crypto v0.39.0\n)\n\nreplace forst => " + forstModule + "\n"
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	download := exec.Command("go", "mod", "download", "golang.org/x/crypto@v0.39.0")
	download.Dir = forstGomod
	if out, err := download.CombinedOutput(); err != nil {
		t.Fatalf("go mod download: %v\n%s", err, out)
	}
	ftconfig := `{
  "server": {"embedded": true, "port": "6321"},
  "node": {
    "enabled": true,
    "hostMode": true,
    "binary": "node",
    "args": ["scripts/host.mjs"]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(ftconfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "host.mjs"), []byte("// host shim stub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forstDir, "host.ts"), []byte(`export function hostPing(): string { return "ready" }`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainSrc := `package main

import node host "./host"

func main() {
	ready := host.hostPing()
	ensure ready is Ok()
	println("forst:app ready: " + ready)
}
`
	if err := os.WriteFile(filepath.Join(forstDir, "main.ft"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	bcryptSrc := `package bcrypt

import "golang.org/x/crypto/bcrypt"

type ComparePasswordRequest = {
	plainPassword: String,
	passwordHash: String
}

type ComparePasswordResponse = {
	valid: Bool
}

func ComparePassword(input ComparePasswordRequest) {
	hashBytes := []byte(input.passwordHash)
	plainBytes := []byte(input.plainPassword)

	compareErr := bcrypt.CompareHashAndPassword(hashBytes, plainBytes)
	if compareErr is Nil() {
		return { valid: true }
	}
	return { valid: false }
}
`
	if err := os.WriteFile(filepath.Join(forstDir, "bcrypt.ft"), []byte(bcryptSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	moduleFns, err := discovery.CollectInvokeFunctionsFromModule(nil, dir)
	if err != nil {
		t.Fatalf("discover module exports: %v", err)
	}
	var discovered []string
	for _, fn := range moduleFns {
		discovered = append(discovered, fn.Package+"."+fn.Name)
	}
	if !strings.Contains(strings.Join(discovered, ","), "bcrypt.ComparePassword") {
		t.Fatalf("expected bcrypt.ComparePassword in discovery, got %v", discovered)
	}
	c := New(Args{
		Command:            "build",
		FilePath:           filepath.Join(forstDir, "main.ft"),
		PackageRoot:        dir,
		ExportStructFields: true,
		LogLevel:           "error",
	}, nil)
	mainCode, nodeRuntime, invokeCode, extraPkgs, _, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if invokeCode == "" {
		t.Fatal("expected invoke companion with cross-package handlers")
	}
	if nodeRuntime == "" {
		t.Fatal("expected node runtime companion for host mode")
	}
	bcryptPkg, ok := extraPkgs["bcrypt"]
	if !ok || bcryptPkg == "" {
		t.Fatalf("expected extra bcrypt package, got %v", extraPkgs)
	}
	if !strings.Contains(bcryptPkg, "golang.org/x/crypto/bcrypt") {
		t.Fatalf("bcrypt package should import golang.org/x/crypto/bcrypt:\n%s", bcryptPkg)
	}
	for _, want := range []string{
		"forst_invoke_bcrypt_ComparePassword",
		"bcrypt.ComparePassword",
		"ForstInvokeWaitForShutdown",
		"invokeembed.MustStartEmbedded",
	} {
		if !strings.Contains(invokeCode, want) {
			t.Fatalf("missing %q in invoke companion:\n%s", want, invokeCode)
		}
	}
	if !strings.Contains(mainCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("main should call shutdown when companion present:\n%s", mainCode)
	}
}

func forstCompilerModuleRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
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
