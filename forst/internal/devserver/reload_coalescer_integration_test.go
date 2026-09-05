package devserver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func TestWatchRuntimeDev_burstFileChanges_coalescesReloads(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ft")
	writeEntry(t, dir, "main.ft", "package main\nfunc main() {}\n")

	var compileCount atomic.Int32
	// Gate reloads: initial compile is allowed; further compiles wait until the test releases.
	blockCompile := make(chan struct{}, 1)
	blockCompile <- struct{}{}
	releaseCompile := make(chan struct{})
	var releaseOnce sync.Once
	releaseAll := func() {
		releaseOnce.Do(func() { close(releaseCompile) })
	}

	deps := stubReloadHooks(RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(_, _, _ string, _ map[string]string, _ map[string]string, boundary string) (string, error) {
			compileCount.Add(1)
			select {
			case <-blockCompile:
			case <-releaseCompile:
			}
			return filepath.Join(boundary, "out.go"), nil
		},
		StartProgram: func(string, string) (*runningChild, error) {
			return &runningChild{stop: func() error { return nil }}, nil
		},
	})

	log := logrus.New()
	log.SetOutput(io.Discard)
	startWatchRuntimeDev(t, log, dir, mainPath, &ftconfig.Config{Dev: ftconfig.DevConfig{AutoRestart: true}}, deps)
	// LIFO cleanups: release blocked CreateOutput before stop so the coalescer
	// goroutine cannot keep writing into t.TempDir after the watch exits.
	t.Cleanup(releaseAll)

	deadline := time.Now().Add(2 * time.Second)
	for compileCount.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if compileCount.Load() < 1 {
		t.Fatal("expected initial compile")
	}

	for i := 0; i < 8; i++ {
		src := fmt.Sprintf("package main\nfunc main() { println(\"v%d\") }\n", i)
		if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Millisecond)
	}

	blockCompile <- struct{}{}
	blockCompile <- struct{}{}

	deadline = time.Now().Add(3 * time.Second)
	for compileCount.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	got := compileCount.Load()
	if got > 3 {
		t.Fatalf("compileCount=%d want at most 3 (initial + reload + one coalesced follow-up)", got)
	}
	if got < 2 {
		t.Fatalf("compileCount=%d want at least 2 reload cycles", got)
	}
}
