package devserver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	restore := stubInvokeReadyWaiter(t)
	defer restore()

	var compileCount atomic.Int32
	blockCompile := make(chan struct{}, 1)
	blockCompile <- struct{}{}

	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(main, _, _ string, _ map[string]string, _ map[string]string, boundary string) (string, error) {
			compileCount.Add(1)
			<-blockCompile
			return filepath.Join(boundary, "out.go"), nil
		},
		StartProgram: func(string, string) (*runningChild, error) {
			return &runningChild{stop: func() error { return nil }}, nil
		},
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	go func() {
		_ = WatchRuntimeDev(log, dir, mainPath, &ftconfig.Config{Dev: ftconfig.DevConfig{AutoRestart: true}}, deps)
	}()

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
