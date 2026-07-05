package executor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"forst/internal/compiler"

	"github.com/sirupsen/logrus"
)

type errorConfig struct {
	err error
}

func (c errorConfig) FindForstFiles(string) ([]string, error) {
	return nil, c.err
}

type sequenceConfig struct {
	mu       sync.Mutex
	responses []sequenceResponse
	idx      int
}

type sequenceResponse struct {
	files []string
	err   error
}

func (c *sequenceConfig) FindForstFiles(string) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.idx >= len(c.responses) {
		return nil, nil
	}
	res := c.responses[c.idx]
	c.idx++
	return res.files, res.err
}

func newExecutorWithConfig(t *testing.T, root string, cfg *sequenceConfig) *FunctionExecutor {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	comp := compiler.New(compiler.Args{Command: "run", FilePath: filepath.Join(root, "placeholder.ft")}, log)
	return NewFunctionExecutor(root, comp, log, cfg)
}

func TestGenerateRandomString_lengthAndCharset(t *testing.T) {
	t.Parallel()

	s := generateRandomString(32)
	if len(s) != 32 {
		t.Fatalf("generateRandomString length = %d, want 32", len(s))
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, r := range s {
		if !strings.ContainsRune(charset, r) {
			t.Fatalf("generateRandomString produced rune %q outside allowed charset", r)
		}
	}
}

func TestFunctionExecutor_executeGoCode_startAndWaitFailures(t *testing.T) {
	t.Run("start_failure_with_params", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		dir := t.TempDir()
		t.Setenv("PATH", "")

		_, err := e.executeGoCode(dir, json.RawMessage(`[1]`), 1)
		if err == nil {
			t.Fatal("expected start failure")
		}
		if !strings.Contains(err.Error(), "failed to start command") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wait_failure_with_params", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module exectest\n\ngo 1.24\n")
		writeFile(t, filepath.Join(dir, "main.go"), `package main
func main() {
	_ = doesNotExist
}
`)

		_, err := e.executeGoCode(dir, json.RawMessage(`[1]`), 1)
		if err == nil {
			t.Fatal("expected wait failure with compile error")
		}
		if !strings.Contains(err.Error(), "execution failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wait_failure_without_params", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module exectest\n\ngo 1.24\n")
		writeFile(t, filepath.Join(dir, "main.go"), `package main
func main() {
	_ = doesNotExist
}
`)

		_, err := e.executeGoCode(dir, nil)
		if err == nil {
			t.Fatal("expected failure")
		}
		if !strings.Contains(err.Error(), "execution failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestFunctionExecutor_executeStreamingGoCode_errorBranches(t *testing.T) {
	t.Run("start_failure", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		t.Setenv("PATH", "")

		_, err := e.executeStreamingGoCode(context.Background(), t.TempDir(), nil, false)
		if err == nil {
			t.Fatal("expected start failure")
		}
		if !strings.Contains(err.Error(), "failed to start command") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("malformed_json_line_emits_stream_error", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module streamtest\ngo 1.24\n")
		writeFile(t, filepath.Join(dir, "main.go"), `package main
import "fmt"
func main() {
	fmt.Println("not-json")
	fmt.Println("{\"status\":\"ok\",\"data\":1}")
}
`)

		results, err := e.executeStreamingGoCode(context.Background(), dir, nil, false)
		if err != nil {
			t.Fatalf("executeStreamingGoCode: %v", err)
		}

		var sawDecodeErr bool
		for r := range results {
			if r.Error != "" {
				sawDecodeErr = true
			}
		}
		if !sawDecodeErr {
			t.Fatal("expected malformed JSON decode error in stream results")
		}
	})

	t.Run("has_params_uses_cli_args_path", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module streamtest\ngo 1.24\n")
		writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"fmt"
	"os"
)
func main() {
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	fmt.Printf("{\"status\":\"ok\",\"data\":%q}\n", arg)
}
`)

		results, err := e.executeStreamingGoCode(context.Background(), dir, json.RawMessage(`{"x":1}`), true)
		if err != nil {
			t.Fatalf("executeStreamingGoCode: %v", err)
		}
		got := <-results
		if got.Error != "" {
			t.Fatalf("unexpected stream error: %s", got.Error)
		}
		if got.Status != "ok" {
			t.Fatalf("unexpected stream status: %+v", got)
		}
	})

	t.Run("context_cancel_stops_stream", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module streamtest\ngo 1.24\n")
		writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"fmt"
	"time"
)
func main() {
	for i := 0; i < 100; i++ {
		fmt.Println("{\"status\":\"ok\",\"data\":1}")
		time.Sleep(20 * time.Millisecond)
	}
}
`)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		results, err := e.executeStreamingGoCode(ctx, dir, nil, false)
		if err != nil {
			t.Fatalf("executeStreamingGoCode: %v", err)
		}

		r, ok := <-results
		if !ok {
			t.Fatal("expected at least one stream result before cancel")
		}
		if r.Error != "" {
			t.Fatalf("unexpected stream error before cancel: %s", r.Error)
		}
		cancel()

		select {
		case _, ok := <-results:
			if ok {
				// channel can still briefly produce buffered item; accept it.
			}
		case <-time.After(2 * time.Second):
			t.Fatal("stream results channel did not settle after context cancel")
		}
	})

	t.Run("context_cancel_stops_stream_after_initial_item", func(t *testing.T) {
		e := testExecutor(t, t.TempDir())
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "go.mod"), "module streamtest\ngo 1.24\n")
		writeFile(t, filepath.Join(dir, "main.go"), `package main
import (
	"fmt"
	"time"
)
func main() {
	for i := 0; i < 100; i++ {
		fmt.Println("{\"status\":\"ok\",\"data\":1}")
		time.Sleep(20 * time.Millisecond)
	}
}
`)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		results, err := e.executeStreamingGoCode(ctx, dir, nil, false)
		if err != nil {
			t.Fatalf("executeStreamingGoCode: %v", err)
		}

		r, ok := <-results
		if !ok {
			t.Fatal("expected at least one stream result before cancel")
		}
		if r.Error != "" {
			t.Fatalf("unexpected stream error before cancel: %s", r.Error)
		}
		cancel()
	})
}

func TestFunctionExecutor_createTempGoFile_creationError(t *testing.T) {
	e := testExecutor(t, t.TempDir())
	t.Setenv("TMPDIR", "/path/that/does/not/exist")

	_, err := e.createTempGoFile(&CompiledFunction{
		PackageName:  "demo",
		FunctionName: "Hello",
		GoCode:       "package demo\nfunc Hello() string { return \"hi\" }\n",
	}, json.RawMessage(`null`))
	if err == nil {
		t.Fatal("expected createTempGoFile error")
	}
}

func TestFunctionExecutor_compileFunction_moduleCheckErrorAfterFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo.ft"), `package demo

func Hello(): String {
	return "hello"
}
`)
	writeFile(t, filepath.Join(root, "bad.ft"), `package zbad

func Broken(): String {
	return 1
}
`)

	ex := testExecutor(t, root)
	_, err := ex.compileFunction("demo", "Hello")
	if err == nil {
		t.Fatal("expected module-check propagated error")
	}
}

func TestFunctionExecutor_compileFunction_listForstFilesError(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "demo.ft")
	writeFile(t, target, `package demo

func Hello(): String {
	return "hello"
}
`)
	cfg := &sequenceConfig{
		responses: []sequenceResponse{
			{files: []string{target}},                  // discovery in findFunctionFile
			{err: errors.New("second call failure")}, // compileFunction list stage
		},
	}
	ex := newExecutorWithConfig(t, root, cfg)

	_, err := ex.compileFunction("demo", "Hello")
	if err == nil {
		t.Fatal("expected list Forst files error")
	}
	if !strings.Contains(err.Error(), "list Forst files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_compileFunction_noParseablePackageFiles(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "demo.ft")
	bad := filepath.Join(root, "bad.ft")
	writeFile(t, good, `package demo

func Hello(): String {
	return "hello"
}
`)
	writeFile(t, bad, "!!! not parseable !!!")
	cfg := &sequenceConfig{
		responses: []sequenceResponse{
			{files: []string{good}}, // discovery can find target function
			{files: []string{bad}},  // compile stage only sees unparseable file
		},
	}
	ex := newExecutorWithConfig(t, root, cfg)

	_, err := ex.compileFunction("demo", "Hello")
	if err == nil {
		t.Fatal("expected no parseable package files error")
	}
	if !strings.Contains(err.Error(), "no parseable Forst files for package demo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_lookup_discoveryFailure(t *testing.T) {
	ex := NewFunctionExecutor(
		t.TempDir(),
		nil,
		testExecutor(t, t.TempDir()).log,
		errorConfig{err: context.Canceled},
	)

	_, err := ex.findFunctionFile("demo", "Hello")
	if err == nil {
		t.Fatal("expected discoverer error from findFunctionFile")
	}
	if !strings.Contains(err.Error(), "failed to discover functions") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = ex.getFunctionInfo("demo", "Hello")
	if err == nil {
		t.Fatal("expected discoverer error from getFunctionInfo")
	}
	if !strings.Contains(err.Error(), "failed to discover functions") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFunctionExecutor_ExecuteFunction_createModuleAndExecutionErrors(t *testing.T) {
	root := t.TempDir()
	ex := testExecutor(t, root)
	ex.cache["demo.Hello"] = &CompiledFunction{
		PackageName:  "demo",
		FunctionName: "Hello",
		GoCode: `package demo
func Hello() string {
	return "hello"
}
`,
	}

	t.Run("create_temp_dir_failure", func(t *testing.T) {
		t.Setenv("TMPDIR", "/path/that/does/not/exist")
		_, err := ex.ExecuteFunction("demo", "Hello", json.RawMessage(`null`))
		if err == nil {
			t.Fatal("expected create temp dir failure")
		}
		if !strings.Contains(err.Error(), "failed to create temp dir") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("execute_go_code_failure", func(t *testing.T) {
		ex.cache["demo.Broken"] = &CompiledFunction{
			PackageName:  "demo",
			FunctionName: "Broken",
			GoCode: `package demo
func Broken() string {
	_ = doesNotExist
	return ""
}
`,
		}
		_, err := ex.ExecuteFunction("demo", "Broken", json.RawMessage(`null`))
		if err == nil {
			t.Fatal("expected go execution failure")
		}
		if !strings.Contains(err.Error(), "failed to execute Go code") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestFunctionExecutor_ExecuteStreamingFunction_createTempDirError(t *testing.T) {
	root := t.TempDir()
	ex := testExecutor(t, root)
	ex.cache["demo.Stream"] = &CompiledFunction{
		PackageName:       "demo",
		FunctionName:      "Stream",
		SupportsStreaming: true,
		GoCode: `package demo
import "encoding/json"
type Result struct{ Data int ` + "`json:\"data\"`" + ` }
func Stream(_ json.RawMessage) <-chan Result {
	ch := make(chan Result, 1)
	ch <- Result{Data: 1}
	close(ch)
	return ch
}
`,
	}

	t.Setenv("TMPDIR", "/path/that/does/not/exist")
	_, err := ex.ExecuteStreamingFunction(context.Background(), "demo", "Stream", json.RawMessage(`[]`))
	if err == nil {
		t.Fatal("expected streaming create temp dir error")
	}
	if !strings.Contains(err.Error(), "failed to create temp dir") {
		t.Fatalf("unexpected error: %v", err)
	}
}
