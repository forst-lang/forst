package devserver

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

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

// newTestLogCapture returns a logger and a thread-safe snapshot of logged lines.
func newTestLogCapture(level logrus.Level) (*logrus.Logger, func() string) {
	var (
		mu     sync.Mutex
		lines  []string
		format = &logrus.TextFormatter{DisableTimestamp: true}
	)
	log := logrus.New()
	log.SetLevel(level)
	log.SetOutput(io.Discard)
	log.AddHook(&testLogHook{callback: func(entry *logrus.Entry) {
		b, err := format.Format(entry)
		if err != nil {
			return
		}
		mu.Lock()
		lines = append(lines, string(b))
		mu.Unlock()
	}})
	snapshot := func() string {
		mu.Lock()
		defer mu.Unlock()
		return strings.Join(lines, "")
	}
	return log, snapshot
}

func stubReloadHooks(deps RuntimeRunDeps) RuntimeRunDeps {
	if deps.InvokeReadyWait == nil {
		deps.InvokeReadyWait = func(string, string, <-chan error, time.Duration) error { return nil }
	}
	if deps.FindInvokePort == nil {
		deps.FindInvokePort = func(_, preferred string) (string, error) { return preferred, nil }
	}
	return deps
}

// startWatchRuntimeDev runs WatchRuntimeDev in the background and registers a
// t.Cleanup that stops it via deps.StopCh and waits for the goroutine to
// exit. Without this, WatchRuntimeDev's watch loop (fsnotify watcher, signal
// handler, etc.) leaks for the remaining lifetime of the test binary, which
// causes flaky cross-test interference (stray log output, contended invoke
// ports) in later tests.
func startWatchRuntimeDev(t *testing.T, log *logrus.Logger, boundaryRoot, entryPath string, cfg *ftconfig.Config, deps RuntimeRunDeps) <-chan error {
	t.Helper()
	stopCh := make(chan struct{})
	deps.StopCh = stopCh
	done := make(chan error, 1)
	go func() {
		done <- WatchRuntimeDev(log, boundaryRoot, entryPath, cfg, deps)
	}()
	t.Cleanup(func() {
		close(stopCh)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Log("WatchRuntimeDev did not exit within cleanup timeout")
		}
	})
	return done
}
