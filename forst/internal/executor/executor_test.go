package executor

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/cmd/forst/compiler"

	"github.com/sirupsen/logrus"
)

// walkForstFilesConfig finds .ft files under rootDir (same behavior as production discovery needs).
type walkForstFilesConfig struct{}

func (walkForstFilesConfig) FindForstFiles(rootDir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".ft") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func testExecutor(t *testing.T, root string) *FunctionExecutor {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	comp := compiler.New(compiler.Args{Command: "run", FilePath: filepath.Join(root, "placeholder.ft")}, log)
	return NewFunctionExecutor(root, comp, log, walkForstFilesConfig{})
}

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
	// Valid parse; typechecker should reject String function returning integer literal.
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
