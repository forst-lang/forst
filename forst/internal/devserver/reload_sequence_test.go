package devserver

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func TestPerformDevReload_compilesBeforeChildStop(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(entry, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var (
		mu     sync.Mutex
		events []string
	)
	record := func(name string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	}

	stopped := make(chan struct{}, 1)
	child := &runningChild{
		stop: func() error {
			record("stop")
			stopped <- struct{}{}
			return nil
		},
		exited: make(chan error),
	}

	origInvokeReady := invokeReadyWaiter
	origPortPick := findNextFreeInvokePortFn
	t.Cleanup(func() {
		invokeReadyWaiter = origInvokeReady
		findNextFreeInvokePortFn = origPortPick
	})
	invokeReadyWaiter = func(string, string, <-chan error, time.Duration) error {
		return nil
	}
	findNextFreeInvokePortFn = func(_, preferred string) (string, error) { return preferred, nil }

	log := logrus.New()
	log.SetOutput(io.Discard)
	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(string, string, string, map[string]string, map[string]string, string) (string, error) {
			record("compile")
			raw, err := os.ReadFile(reloadingMarkerPath(dir))
			if err != nil || len(raw) == 0 {
				t.Fatal("reload marker should be set during compile")
			}
			select {
			case <-stopped:
				t.Fatal("child stopped before compile finished")
			default:
			}
			return filepath.Join(dir, "out.go"), nil
		},
		StartProgram: func(string, string) (*runningChild, error) {
			record("start")
			return &runningChild{
				stop:   func() error { return nil },
				exited: make(chan error),
			}, nil
		},
	}

	performDevReload(reloadParams{
		log:          log,
		boundaryRoot: dir,
		entryPath:    entry,
		cfg:          &ftconfig.Config{},
		deps:         deps,
		autoRestart:  true,
		invokeAddr:   "127.0.0.1:1",
		healthURL:    "http://127.0.0.1:1/health",
		gen:          1,
		child:        child,
	})

	mu.Lock()
	got := append([]string(nil), events...)
	mu.Unlock()
	want := []string{"compile", "stop", "start"}
	if len(got) != len(want) {
		t.Fatalf("events=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events=%v want %v", got, want)
		}
	}
}
