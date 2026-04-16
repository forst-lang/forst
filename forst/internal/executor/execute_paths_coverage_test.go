package executor

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestFunctionExecutor_executeGoCode_withAndWithoutParams(t *testing.T) {
	t.Parallel()
	e := testExecutor(t, t.TempDir())
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module exectest\ngo 1.24\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"fmt"
	"io"
	"os"
)
func main() {
	_, _ = io.ReadAll(os.Stdin)
	fmt.Print("ok")
}
`)

	outNoParams, err := e.executeGoCode(dir, nil)
	if err != nil {
		t.Fatalf("executeGoCode without params: %v", err)
	}
	if outNoParams == "" {
		t.Fatal("expected output without params")
	}

	outWithParams, err := e.executeGoCode(dir, json.RawMessage(`null`), 1)
	if err != nil {
		t.Fatalf("executeGoCode with params: %v", err)
	}
	if outWithParams == "" {
		t.Fatal("expected output with params")
	}
}

func TestFunctionExecutor_ExecuteStreamingFunction_successFromCache(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ex := testExecutor(t, root)
	ex.cache["demo.Stream"] = &CompiledFunction{
		PackageName:       "demo",
		FunctionName:      "Stream",
		SupportsStreaming: true,
		GoCode: `package demo
import "encoding/json"
type Result struct {
	Status string ` + "`json:\"status\"`" + `
	Data   int    ` + "`json:\"data\"`" + `
}
func Stream(_ json.RawMessage) <-chan Result {
	ch := make(chan Result, 1)
	ch <- Result{Status: "ok", Data: 1}
	close(ch)
	return ch
}
`,
	}

	results, err := ex.ExecuteStreamingFunction(context.Background(), "demo", "Stream", json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("ExecuteStreamingFunction: %v", err)
	}
	for r := range results {
		if r.Error != "" {
			t.Fatalf("unexpected stream error: %s", r.Error)
		}
	}
}

