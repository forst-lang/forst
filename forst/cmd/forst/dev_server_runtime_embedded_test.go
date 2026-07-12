package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"
	"forst/internal/devserver"
	"forst/internal/invokeserver"

	"github.com/sirupsen/logrus"
)

func TestHelper_devRuntimeEmbedded(t *testing.T) {
	if os.Getenv("FORST_DEV_RUNTIME_EMBEDDED") != "1" {
		return
	}
	root := os.Getenv("FORST_DEV_RUNTIME_EMBEDDED_ROOT")
	os.Args = []string{"forst", "dev", "-root", root, "-entry", "main.ft", "-log-level", "error"}
	os.Exit(runMain(os.Args))
}

func TestRunMain_dev_runtime_missingEntry_exitsNonZero(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runMain([]string{"forst", "dev", "-root", dir, "-config", filepath.Join(dir, "ftconfig.json"), "-log-level", "error"}); code != 1 {
		t.Fatalf("want exit 1, got %d", code)
	}
}

func TestDevRuntime_embeddedCompile_buildsProgram(t *testing.T) {
	dir := t.TempDir()
	mainSrc := `package main

type EchoRequest = { message: String }
type EchoResponse = { echo: String, timestamp: Int }

func Echo(input EchoRequest) {
  return { echo: input.message, timestamp: 42 }
}

func main() {
  println("embedded invoke listening")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true,"port":"0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	cfg, err := LoadConfig(filepath.Join(dir, "ftconfig.json"))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := devserver.ResolveEntry(dir, &cfg.Config, "main.ft")
	if err != nil {
		t.Fatal(err)
	}

	var mainCode, invokeCode string
	deps := devserver.RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(main, nodert, invoke string, extra map[string]string, extraImports map[string]string, boundary string) (string, error) {
			mainCode = main
			invokeCode = invoke
			return compiler.CreateTempOutputFiles(main, nodert, invoke, extra, extraImports, boundary)
		},
		RunProgram: func(string, string) error { return nil },
	}
	if err := devserver.RunRuntimeDev(log, dir, entry, &cfg.Config, deps); err != nil {
		t.Fatal(err)
	}
	if invokeCode == "" {
		t.Fatal("expected invoke companion for embedded config")
	}
	if !strings.Contains(mainCode, "ForstInvokeWaitForShutdown") {
		t.Fatalf("main missing shutdown hook:\n%s", mainCode)
	}
	if err := compiler.BuildGoProgram(mainCode, invokeCode, ""); err != nil {
		t.Fatalf("BuildGoProgram: %v", err)
	}
}

func TestDevRuntime_embeddedInvokeReady_inProcess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FORST_BOUNDARY_ROOT", dir)
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true,"port":"0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if invokeserver.GlobalRegistry() != nil {
			invokeserver.NotifyShutdown()
		}
	})
	invokeserver.MustStartEmbedded()
	ready := filepath.Join(dir, ".forst", "invoke.ready")
	raw, err := os.ReadFile(ready)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "embedded") {
		t.Fatalf("unexpected ready payload: %s", raw)
	}
}
