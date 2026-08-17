package main

import (
	"io"
	"path/filepath"
	"sync/atomic"
	"testing"

	"forst/internal/devserver"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func TestRunRuntimeDevFn_runsStartupGenerate(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	cfg := DefaultConfig()
	log := logrus.New()
	log.SetOutput(io.Discard)

	var calls atomic.Int32
	origGen := runtimeDevGenerateFn
	runtimeDevGenerateFn = func(boundaryRoot string, c *ForstConfig, l *logrus.Logger) error {
		if boundaryRoot != dir || c != cfg {
			t.Fatalf("generate boundary=%q cfg=%p", boundaryRoot, c)
		}
		calls.Add(1)
		return nil
	}
	t.Cleanup(func() { runtimeDevGenerateFn = origGen })

	origRun := runRuntimeDevEntry
	runRuntimeDevEntry = func(*logrus.Logger, string, string, *ftconfig.Config, devserver.RuntimeRunDeps) error {
		return nil
	}
	t.Cleanup(func() { runRuntimeDevEntry = origRun })

	if err := runRuntimeDevFn(log, dir, entry, cfg); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("startup generate calls = %d want 1", calls.Load())
	}
}

func TestWatchRuntimeDevFn_skipsStartupGenerate(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	cfg := DefaultConfig()
	log := logrus.New()
	log.SetOutput(io.Discard)

	var calls atomic.Int32
	origGen := runtimeDevGenerateFn
	runtimeDevGenerateFn = func(string, *ForstConfig, *logrus.Logger) error {
		calls.Add(1)
		return nil
	}
	t.Cleanup(func() { runtimeDevGenerateFn = origGen })

	origWatch := watchRuntimeDevEntry
	watchRuntimeDevEntry = func(*logrus.Logger, string, string, *ftconfig.Config, devserver.RuntimeRunDeps) error {
		return nil
	}
	t.Cleanup(func() { watchRuntimeDevEntry = origWatch })

	if err := watchRuntimeDevFn(log, dir, entry, cfg); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("watch runtime startup generate calls = %d want 0", calls.Load())
	}
}
