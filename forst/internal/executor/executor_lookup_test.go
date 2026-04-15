package executor

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestFunctionExecutor_getOrCompileFunction_cacheMissThenHit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hello.ft"), `package demo

func Hello(): String {
	return "Hello"
}
`)
	ex := testExecutor(t, root)

	first, err := ex.getOrCompileFunction("demo", "Hello")
	if err != nil {
		t.Fatalf("first getOrCompileFunction: %v", err)
	}
	if first == nil {
		t.Fatal("expected compiled function on cache miss")
	}
	if first.FunctionName != "Hello" || first.PackageName != "demo" {
		t.Fatalf("unexpected compiled function metadata: %+v", first)
	}

	second, err := ex.getOrCompileFunction("demo", "Hello")
	if err != nil {
		t.Fatalf("second getOrCompileFunction: %v", err)
	}
	if second == nil {
		t.Fatal("expected compiled function on cache hit")
	}
	if first != second {
		t.Fatal("expected cache hit to return same compiled function pointer")
	}

	ex.mu.RLock()
	defer ex.mu.RUnlock()
	if len(ex.cache) != 1 {
		t.Fatalf("expected single cache entry, got %d", len(ex.cache))
	}
	if _, ok := ex.cache["demo.Hello"]; !ok {
		t.Fatalf("expected cache key demo.Hello, got keys: %+v", ex.cache)
	}
}

func TestFunctionExecutor_getOrCompileFunction_compileErrorPropagates(t *testing.T) {
	ex := testExecutor(t, t.TempDir())
	compiled, err := ex.getOrCompileFunction("missingpkg", "NoSuchFn")
	if err == nil {
		t.Fatal("expected compile/get error")
	}
	if compiled != nil {
		t.Fatalf("expected nil compiled function on error, got %+v", compiled)
	}
}

func TestFunctionExecutor_findFunctionFile_successAndMissingCases(t *testing.T) {
	root := t.TempDir()
	goodPath := filepath.Join(root, "hello.ft")
	writeFile(t, goodPath, `package demo

func Hello(): String {
	return "Hello"
}
`)
	ex := testExecutor(t, root)

	foundPath, err := ex.findFunctionFile("demo", "Hello")
	if err != nil {
		t.Fatalf("findFunctionFile success case: %v", err)
	}
	if foundPath != goodPath {
		t.Fatalf("expected file path %q, got %q", goodPath, foundPath)
	}

	_, err = ex.findFunctionFile("missingpkg", "Hello")
	if err == nil || !strings.Contains(err.Error(), "package missingpkg not found") {
		t.Fatalf("expected missing package error, got %v", err)
	}

	_, err = ex.findFunctionFile("demo", "NoSuchFn")
	if err == nil || !strings.Contains(err.Error(), "function NoSuchFn not found in package demo") {
		t.Fatalf("expected missing function error, got %v", err)
	}
}

func TestFunctionExecutor_getFunctionInfo_successAndMissingCases(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hello.ft"), `package demo

func Hello(input String): String {
	return input
}
`)
	ex := testExecutor(t, root)

	fnInfo, err := ex.getFunctionInfo("demo", "Hello")
	if err != nil {
		t.Fatalf("getFunctionInfo success case: %v", err)
	}
	if fnInfo == nil {
		t.Fatal("expected function info, got nil")
	}
	if fnInfo.Name != "Hello" || fnInfo.Package != "demo" {
		t.Fatalf("unexpected function info metadata: %+v", fnInfo)
	}

	_, err = ex.getFunctionInfo("missingpkg", "Hello")
	if err == nil || !strings.Contains(err.Error(), "package missingpkg not found") {
		t.Fatalf("expected missing package error, got %v", err)
	}

	_, err = ex.getFunctionInfo("demo", "NoSuchFn")
	if err == nil || !strings.Contains(err.Error(), "function NoSuchFn not found in package demo") {
		t.Fatalf("expected missing function error, got %v", err)
	}
}

func TestFunctionExecutor_cacheReuseAcrossExecuteAndLookup(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hello.ft"), `package demo

func Hello(): String {
	return "Hello"
}
`)
	ex := testExecutor(t, root)

	_, err := ex.ExecuteFunction("demo", "Hello", json.RawMessage("null"))
	if err != nil {
		t.Fatalf("ExecuteFunction: %v", err)
	}

	fromLookup, err := ex.getOrCompileFunction("demo", "Hello")
	if err != nil {
		t.Fatalf("getOrCompileFunction after ExecuteFunction: %v", err)
	}
	if fromLookup == nil {
		t.Fatal("expected cached compiled function")
	}

	ex.mu.RLock()
	cached := ex.cache["demo.Hello"]
	ex.mu.RUnlock()
	if cached == nil {
		t.Fatal("expected cache entry demo.Hello after ExecuteFunction")
	}
	if cached != fromLookup {
		t.Fatal("expected getOrCompileFunction to return same cached pointer after ExecuteFunction")
	}
}
