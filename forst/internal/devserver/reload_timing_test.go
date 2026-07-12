package devserver

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"forst/internal/compiler"
	"forst/internal/ftconfig"
	"forst/internal/modulecheck"

	"github.com/sirupsen/logrus"
)

func TestPerformDevReload_emitsReloadTiming(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(entry, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var (
		mu          sync.Mutex
		events      []string
		fields      []logrus.Fields
		startedPath string
	)
	record := func(name string) {
		mu.Lock()
		events = append(events, name)
		mu.Unlock()
	}

	child := &runningChild{
		stop: func() error {
			record("stop")
			return nil
		},
		exited: make(chan error),
	}

	t.Cleanup(func() {
		modulecheck.ResetPassCountForTest()
	})

	log := logrus.New()
	log.SetOutput(io.Discard)
	log.AddHook(&testLogHook{callback: func(entry *logrus.Entry) {
		if entry.Data["event"] == nil {
			return
		}
		ev, _ := entry.Data["event"].(string)
		if ev != "reload_timing_start" && ev != "reload_timing" {
			return
		}
		mu.Lock()
		cp := make(logrus.Fields, len(entry.Data))
		for k, v := range entry.Data {
			cp[k] = v
		}
		fields = append(fields, cp)
		mu.Unlock()
	}})

	deps := stubReloadHooks(RuntimeRunDeps{
		NewCompiler: func(args compiler.Args, l *logrus.Logger) *compiler.Compiler {
			if !args.ReloadProfile {
				t.Fatal("expected ReloadProfile enabled for watch reload")
			}
			return compiler.New(args, l)
		},
		CreateOutputForReload: func(string, string, string, map[string]string, map[string]string, string, *compiler.CompileSandboxTiming) (string, error) {
			record("compile")
			return filepath.Join(dir, "out.go"), nil
		},
		BuildProgram: func(string, string, string) error { return nil },
		StartProgram: func(outputPath, boundaryRoot string) (*runningChild, error) {
			record("start")
			mu.Lock()
			startedPath = outputPath
			mu.Unlock()
			return &runningChild{
				stop:   func() error { return nil },
				exited: make(chan error),
			}, nil
		},
	})

	cfg := &ftconfig.Config{Dev: ftconfig.DevConfig{HotReload: true}}
	performDevReload(reloadParams{
		log:          log,
		boundaryRoot: dir,
		entryPath:    entry,
		cfg:          cfg,
		deps:         deps,
		autoRestart:  true,
		invokeAddr:   "127.0.0.1:1",
		healthURL:    "http://127.0.0.1:1/health",
		gen:          1,
		child:        child,
	})

	mu.Lock()
	gotEvents := append([]string(nil), events...)
	gotFields := append([]logrus.Fields(nil), fields...)
	mu.Unlock()

	wantEvents := []string{"compile", "stop", "start"}
	if len(gotEvents) != len(wantEvents) {
		t.Fatalf("events=%v want %v", gotEvents, wantEvents)
	}

	var startFields, completeFields logrus.Fields
	for _, f := range gotFields {
		switch f["event"] {
		case "reload_timing_start":
			startFields = f
		case "reload_timing":
			completeFields = f
		}
	}
	if startFields == nil {
		t.Fatal("missing reload_timing_start log")
	}
	if completeFields == nil {
		t.Fatal("missing reload_timing log")
	}
	if startFields["generation"] != uint64(1) {
		t.Fatalf("reload_timing_start generation=%v want 1", startFields["generation"])
	}
	for _, key := range []string{
		"mark_reloading_ms", "forst_compile_ms", "go_build_ms", "child_stop_ms",
		"port_pick_ms", "go_start_ms", "invoke_ready_ms", "total_ms", "modulecheck_passes",
	} {
		if _, ok := completeFields[key]; !ok {
			t.Fatalf("reload_timing missing field %q: %v", key, completeFields)
		}
	}
	if completeFields["success"] != true {
		t.Fatalf("reload_timing success=%v want true", completeFields["success"])
	}
	wantBin := compiler.DevBinPath(dir)
	mu.Lock()
	gotStarted := startedPath
	mu.Unlock()
	if gotStarted != wantBin {
		t.Fatalf("StartProgram path=%q want dev bin %q", gotStarted, wantBin)
	}
}

type testLogHook struct {
	callback func(*logrus.Entry)
}

func (h *testLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *testLogHook) Fire(entry *logrus.Entry) error {
	if h.callback != nil {
		h.callback(entry)
	}
	return nil
}

func TestReloadProfileEnabled(t *testing.T) {
	t.Setenv(envReloadTrace, "")
	if reloadProfileEnabled(&ftconfig.Config{Dev: ftconfig.DevConfig{HotReload: true}}) {
		// ok
	} else {
		t.Fatal("expected hotReload to enable profile")
	}
	if reloadProfileEnabled(&ftconfig.Config{}) {
		t.Fatal("expected profile disabled without watch/hotReload")
	}
	t.Setenv(envReloadTrace, "1")
	if !reloadProfileEnabled(nil) {
		t.Fatal("expected FORST_RELOAD_TRACE=1 to enable profile")
	}
}

func TestReloadTiming_logComplete(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	rt := newReloadTiming()
	rt.recordMarkReloading(time.Now())
	rt.logComplete(log, 2, true)

	if !strings.Contains(buf.String(), "reload complete") {
		t.Fatalf("log output=%q", buf.String())
	}
	if !strings.Contains(buf.String(), "reload_timing") {
		t.Fatalf("log output=%q", buf.String())
	}
}
