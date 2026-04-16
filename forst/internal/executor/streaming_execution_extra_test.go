package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFunctionExecutor_createStreamingTempGoFile_createsModule(t *testing.T) {
	t.Parallel()
	e := testExecutor(t, t.TempDir())
	cf := &CompiledFunction{
		PackageName:       "main",
		FunctionName:      "Run",
		GoCode:            "package main\nfunc Run() int { return 1 }\n",
		SupportsStreaming: true,
	}
	dir, err := e.createStreamingTempGoFile(cf, json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("createStreamingTempGoFile: %v", err)
	}
	defer os.RemoveAll(dir)
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err != nil {
		t.Fatalf("expected generated main.go: %v", err)
	}
}

func TestFunctionExecutor_executeStreamingGoCode_readsJSONLines(t *testing.T) {
	t.Parallel()
	e := testExecutor(t, t.TempDir())
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module streamtest\ngo 1.24\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main
import "fmt"
func main() {
	fmt.Println("{\"status\":\"ok\",\"data\":123}")
}
`)
	results, err := e.executeStreamingGoCode(context.Background(), dir, nil, false)
	if err != nil {
		t.Fatalf("executeStreamingGoCode: %v", err)
	}
	gotAny := false
	for r := range results {
		gotAny = true
		if r.Error != "" {
			t.Fatalf("unexpected streaming error: %s", r.Error)
		}
	}
	if !gotAny {
		t.Fatal("expected at least one streaming result")
	}
}

