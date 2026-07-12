package compiler

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestCompilePhaseTiming_emittedWhenReloadProfile(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(entry, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var captured logrus.Fields
	log := logrus.New()
	log.SetOutput(io.Discard)
	log.AddHook(&compileTimingHook{callback: func(f logrus.Fields) {
		captured = f
	}})

	c := New(Args{
		Command:       "run",
		FilePath:      entry,
		PackageRoot:   dir,
		ReloadProfile: true,
	}, log)

	if _, _, _, _, _, err := c.CompileWithNodeRuntime(); err != nil {
		t.Fatalf("CompileWithNodeRuntime: %v", err)
	}
	timings := c.TakeCompileTimings()
	var sandbox CompileSandboxTiming
	if _, err := CreateTempOutputFilesProfiled("package main\nfunc main() {}", "", "", nil, nil, dir, &sandbox); err != nil {
		t.Fatalf("CreateTempOutputFilesProfiled: %v", err)
	}
	timings.WriteSandboxMs = sandbox.WriteSandboxMs
	timings.GoModTidyMs = sandbox.GoModTidyMs
	c.LogCompilePhaseTiming(timings)

	if captured == nil {
		t.Fatal("missing compile_timing log")
	}
	if captured["event"] != "compile_timing" {
		t.Fatalf("event=%v want compile_timing", captured["event"])
	}
	for _, key := range []string{"load_parse_ms", "typecheck_ms", "codegen_ms", "write_sandbox_ms", "go_mod_tidy_ms"} {
		if _, ok := captured[key]; !ok {
			t.Fatalf("compile_timing missing %q", key)
		}
	}
}

type compileTimingHook struct {
	callback func(logrus.Fields)
}

func (h *compileTimingHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *compileTimingHook) Fire(entry *logrus.Entry) error {
	if entry.Data["event"] == "compile_timing" && h.callback != nil {
		cp := make(logrus.Fields, len(entry.Data))
		for k, v := range entry.Data {
			cp[k] = v
		}
		h.callback(cp)
	}
	return nil
}
