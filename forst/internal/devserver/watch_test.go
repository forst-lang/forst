package devserver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func TestRuntimeWatchEnabled(t *testing.T) {
	if RuntimeWatchEnabled(nil) {
		t.Fatal("nil config should not enable watch")
	}
	if RuntimeWatchEnabled(&ftconfig.Config{}) {
		t.Fatal("zero config should not enable watch")
	}
	if !RuntimeWatchEnabled(&ftconfig.Config{Dev: ftconfig.DevConfig{HotReload: true}}) {
		t.Fatal("hotReload should enable watch")
	}
	if !RuntimeWatchEnabled(&ftconfig.Config{Dev: ftconfig.DevConfig{Watch: true}}) {
		t.Fatal("watch should enable watch")
	}
}

func TestWatchPackageRoot_ignoresNonFt(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "main.ft", "package main\n")

	var triggers atomic.Int32
	done := make(chan struct{})
	stop := make(chan struct{})
	go func() {
		_ = WatchPackageRoot(nil, dir, nil, 50*time.Millisecond, func(string) {
			triggers.Add(1)
		}, stop)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if triggers.Load() != 0 {
		t.Fatalf("non-.ft change should not trigger reload, got %d", triggers.Load())
	}

	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\n// edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for triggers.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if triggers.Load() < 1 {
		t.Fatal("expected .ft write to trigger reload")
	}
	close(stop)
	<-done
}

func TestWatchPackageRoot_detectsAtomicRenameSave(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ft")
	writeEntry(t, dir, "main.ft", "package main\n")

	var triggers atomic.Int32
	done := make(chan struct{})
	stop := make(chan struct{})
	go func() {
		_ = WatchPackageRoot(nil, dir, nil, 50*time.Millisecond, func(string) {
			triggers.Add(1)
		}, stop)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)

	tmpPath := mainPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte("package main\n// renamed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmpPath, mainPath); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for triggers.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	close(stop)
	<-done
	if triggers.Load() < 1 {
		t.Fatal("expected atomic rename save to trigger reload")
	}
}

func TestWatchRuntimeDev_compileErrorOnStart_keepsWatching(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "main.ft", "package main\nfunc main() {}\n")
	restore := stubInvokeReadyWaiter(t)
	defer restore()

	log := logrus.New()
	log.SetOutput(io.Discard)
	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(string, string, string, map[string]string, map[string]string, string) (string, error) {
			return "", errors.New("injected compile failure")
		},
		StartProgram: func(string, string) (*runningChild, error) {
			t.Fatal("StartProgram should not run when compile fails")
			return nil, nil
		},
	}

	done := make(chan error, 1)
	go func() {
		done <- WatchRuntimeDev(log, dir, filepath.Join(dir, "main.ft"), nil, deps)
	}()

	select {
	case err := <-done:
		t.Fatalf("watch loop returned early: %v", err)
	case <-time.After(400 * time.Millisecond):
	}
}

func TestWatchRuntimeDev_compileError_logsError(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "main.ft", "package main\nfunc main() {}\n")
	restore := stubInvokeReadyWaiter(t)
	defer restore()

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(string, string, string, map[string]string, map[string]string, string) (string, error) {
			return "", errors.New("type error: injected")
		},
		StartProgram: func(string, string) (*runningChild, error) {
			return nil, errors.New("should not start")
		},
	}

	done := make(chan struct{})
	go func() {
		_ = WatchRuntimeDev(log, dir, filepath.Join(dir, "main.ft"), nil, deps)
		close(done)
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for !strings.Contains(buf.String(), "type error: injected") && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if !strings.Contains(buf.String(), "type error: injected") {
		t.Fatalf("expected compile error in logs, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "Invoke server is down until compilation succeeds on next save") {
		t.Fatalf("expected warn about invoke down, got: %s", buf.String())
	}
}

func TestWatchRuntimeDev_fileChange_recompilesAndRestarts(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ft")
	writeEntry(t, dir, "main.ft", "package main\nfunc main() {}\n")
	restore := stubInvokeReadyWaiter(t)
	defer restore()

	var compileCount atomic.Int32
	var startCount atomic.Int32
	log := logrus.New()
	log.SetOutput(io.Discard)

	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(main, nodert, invoke string, extra map[string]string, _ map[string]string, boundary string) (string, error) {
			compileCount.Add(1)
			if main == "" {
				return "", errors.New("empty main")
			}
			return filepath.Join(boundary, ".forst", "run", "test", "main.go"), nil
		},
		StartProgram: func(string, string) (*runningChild, error) {
			startCount.Add(1)
			return &runningChild{stop: func() error { return nil }}, nil
		},
	}

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

	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() { println(\"v2\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(3 * time.Second)
	for compileCount.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if compileCount.Load() < 2 {
		t.Fatalf("expected recompile after file change, compileCount=%d", compileCount.Load())
	}
	if startCount.Load() < 2 {
		t.Fatalf("expected process restart after file change, startCount=%d", startCount.Load())
	}
}

func TestWatchRuntimeDev_autoRestartFalse_doesNotStartProcess(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ft")
	writeEntry(t, dir, "main.ft", "package main\nfunc main() {}\n")
	restore := stubInvokeReadyWaiter(t)
	defer restore()

	log := logrus.New()
	log.SetOutput(io.Discard)
	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(main, _, _ string, _ map[string]string, _ map[string]string, boundary string) (string, error) {
			return filepath.Join(boundary, "out.go"), nil
		},
		StartProgram: func(string, string) (*runningChild, error) {
			t.Fatal("StartProgram should not run when autoRestart is false")
			return nil, nil
		},
	}

	done := make(chan struct{})
	go func() {
		_ = WatchRuntimeDev(log, dir, mainPath, &ftconfig.Config{Dev: ftconfig.DevConfig{AutoRestart: false}}, deps)
		close(done)
	}()

	time.Sleep(300 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("watch returned early")
	default:
	}
}

func TestCollectWatchDirs_onlyParentsOfForstFiles(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "forst/main.ft", "package main\n")
	writeEntry(t, dir, "forst/pkg/util.ft", "package pkg\n")
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "app", "routes"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := ftconfig.Default()
	dirs, err := collectWatchDirs(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if strings.Contains(d, "node_modules") || strings.Contains(d, "app") {
			t.Fatalf("should only watch .ft parent dirs, got %v", dirs)
		}
	}
	if len(dirs) > 5 {
		t.Fatalf("expected few watch dirs, got %d: %v", len(dirs), dirs)
	}
}

func stubInvokeReadyWaiter(t *testing.T) func() {
	t.Helper()
	origReady := invokeReadyWaiter
	origPortPick := findNextFreeInvokePortFn
	invokeReadyWaiter = func(string, string, <-chan error, time.Duration) error { return nil }
	findNextFreeInvokePortFn = func(_, preferred string) (string, error) { return preferred, nil }
	return func() {
		invokeReadyWaiter = origReady
		findNextFreeInvokePortFn = origPortPick
	}
}

func TestWatchRuntimeDev_reloadStopsBeforeCompile(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ft")
	writeEntry(t, dir, "main.ft", "package main\nfunc main() {}\n")
	restore := stubInvokeReadyWaiter(t)
	defer restore()

	var order []string
	var mu sync.Mutex
	record := func(step string) {
		mu.Lock()
		order = append(order, step)
		mu.Unlock()
	}

	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(main, _, _ string, _ map[string]string, _ map[string]string, boundary string) (string, error) {
			record("compile")
			return filepath.Join(boundary, "out.go"), nil
		},
		StartProgram: func(string, string) (*runningChild, error) {
			record("start")
			return &runningChild{stop: func() error {
				record("stop")
				return nil
			}}, nil
		},
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	go func() {
		_ = WatchRuntimeDev(log, dir, mainPath, &ftconfig.Config{Dev: ftconfig.DevConfig{AutoRestart: true}}, deps)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		ready := len(order) >= 2 && order[0] == "compile" && order[1] == "start"
		mu.Unlock()
		if ready || time.Now().After(deadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() { println(\"v2\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		n := len(order)
		mu.Unlock()
		if n >= 5 || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()
	// initial: compile, start; reload: compile (incl. build) before stop
	want := []string{"compile", "start", "compile", "stop", "start"}
	if len(got) < len(want) {
		t.Fatalf("order=%v want at least %v", got, want)
	}
	for i, step := range want {
		if got[i] != step {
			t.Fatalf("order=%v want prefix %v", got, want)
		}
	}
}

func TestWatchRuntimeDev_childExitBeforeReady_logsFailure(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.ft")
	writeEntry(t, dir, "main.ft", "package main\nfunc main() {}\n")

	origReady := invokeReadyWaiter
	origPortPick := findNextFreeInvokePortFn
	invokeReadyWaiter = WaitForInvokeReady
	findNextFreeInvokePortFn = func(_, preferred string) (string, error) { return preferred, nil }
	t.Cleanup(func() {
		invokeReadyWaiter = origReady
		findNextFreeInvokePortFn = origPortPick
	})

	exited := make(chan error, 1)
	exited <- fmt.Errorf("bind: address already in use")

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.ErrorLevel)

	deps := RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			return compiler.New(args, l)
		},
		CreateOutput: func(main, _, _ string, _ map[string]string, _ map[string]string, boundary string) (string, error) {
			return filepath.Join(boundary, "out.go"), nil
		},
		StartProgram: func(string, string) (*runningChild, error) {
			return &runningChild{
				stop:   func() error { return nil },
				exited: exited,
			}, nil
		},
	}

	go func() {
		_ = WatchRuntimeDev(log, dir, mainPath, &ftconfig.Config{Dev: ftconfig.DevConfig{AutoRestart: true}}, deps)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(buf.String(), "reload failed") && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if !strings.Contains(buf.String(), "reload failed") {
		t.Fatalf("expected reload failure log, got: %s", buf.String())
	}
}
