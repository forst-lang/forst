package executor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestFunctionExecutor_ExecuteFunction_unknownPackage(t *testing.T) {
	root := t.TempDir()
	ex := testExecutor(t, root)

	_, err := ex.ExecuteFunction("missingpkg", "Fn", json.RawMessage("null"))
	if err == nil {
		t.Fatal("expected error for unknown package")
	}
	if !strings.Contains(err.Error(), "failed to get function") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_ExecuteFunction_unknownFunction(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.ft"), `package demo

func Greet(): String {
	return "x"
}
`)
	ex := testExecutor(t, root)

	_, err := ex.ExecuteFunction("demo", "NoSuchFn", json.RawMessage("null"))
	if err == nil {
		t.Fatal("expected error for unknown function")
	}
	if !strings.Contains(err.Error(), "failed to get function") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_ExecuteFunction_typecheckError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "bad.ft"), `package demo

func Broken(): String {
	return 1
}
`)
	ex := testExecutor(t, root)

	_, err := ex.ExecuteFunction("demo", "Broken", json.RawMessage("null"))
	if err == nil {
		t.Fatal("expected error when typecheck fails")
	}
	if !strings.Contains(err.Error(), "failed to get function") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_ExecuteFunction_crossFileAliasReturn(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "types.ft"), `package demo

type Greeting = String
`)
	writeFile(t, filepath.Join(root, "hello.ft"), `package demo

func Hello(): Greeting {
	return "Hello"
}
`)
	ex := testExecutor(t, root)

	res, err := ex.ExecuteFunction("demo", "Hello", json.RawMessage("null"))
	if err != nil {
		t.Fatalf("ExecuteFunction: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if !strings.Contains(res.Output, "Hello") {
		t.Fatalf("expected Hello in output, got Output=%q Result=%s", res.Output, res.Result)
	}
}

func TestFunctionExecutor_ExecuteFunction_crossFileCall(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "lib.ft"), `package demo

func Helper(): String {
	return "Hello"
}
`)
	writeFile(t, filepath.Join(root, "hello.ft"), `package demo

func Hello(): String {
	return Helper()
}
`)
	ex := testExecutor(t, root)

	res, err := ex.ExecuteFunction("demo", "Hello", json.RawMessage("null"))
	if err != nil {
		t.Fatalf("ExecuteFunction: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if !strings.Contains(res.Output, "Hello") {
		t.Fatalf("expected Hello in output, got Output=%q Result=%s", res.Output, res.Result)
	}
}

func TestFunctionExecutor_ExecuteFunction_Greet(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hello.ft"), `package demo

func Greet(): String {
	return "Hello"
}
`)
	ex := testExecutor(t, root)

	res, err := ex.ExecuteFunction("demo", "Greet", json.RawMessage("null"))
	if err != nil {
		t.Fatalf("ExecuteFunction: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if !strings.Contains(res.Output, "Hello") {
		t.Fatalf("expected Hello in output, got Output=%q Result=%s", res.Output, res.Result)
	}
}

func TestFunctionExecutor_ExecuteStreamingFunction_unknownPackage(t *testing.T) {
	root := t.TempDir()
	ex := testExecutor(t, root)

	_, err := ex.ExecuteStreamingFunction(context.Background(), "missingpkg", "Fn", json.RawMessage("null"))
	if err == nil {
		t.Fatal("expected error for unknown package")
	}
	if !strings.Contains(err.Error(), "failed to get function") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_ExecuteStreamingFunction_notStreamingFunction(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hello.ft"), `package demo

func Hello(): String {
	return "Hello"
}
`)
	ex := testExecutor(t, root)

	_, err := ex.ExecuteStreamingFunction(context.Background(), "demo", "Hello", json.RawMessage("null"))
	if err == nil {
		t.Fatal("expected non-streaming function error")
	}
	if !strings.Contains(err.Error(), "does not support streaming") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_ExecuteFunction_skipsBadFileAndUsesGoodPackageFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "good.ft"), `package demo

func Hello(): String {
	return "Hello"
}
`)
	writeFile(t, filepath.Join(root, "bad.ft"), `@@@ this is not valid forst @@`)

	ex := testExecutor(t, root)
	res, err := ex.ExecuteFunction("demo", "Hello", json.RawMessage("null"))
	if err != nil {
		t.Fatalf("ExecuteFunction should succeed despite bad file: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("expected success result, got %+v", res)
	}
	if !strings.Contains(res.Output, "Hello") {
		t.Fatalf("expected Hello output, got Output=%q Result=%s", res.Output, res.Result)
	}
}

func TestFunctionExecutor_ExecuteFunction_allFilesUnparseableReturnsNoParseableError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "bad1.ft"), `@@@ invalid @@@`)
	writeFile(t, filepath.Join(root, "bad2.ft"), `!!! also invalid !!!`)

	ex := testExecutor(t, root)
	_, err := ex.ExecuteFunction("demo", "AnyFn", json.RawMessage("null"))
	if err == nil {
		t.Fatal("expected error when all package files are unparseable")
	}
	if !strings.Contains(err.Error(), "failed to get function") {
		t.Fatalf("expected wrapped get-function error, got %v", err)
	}
}
