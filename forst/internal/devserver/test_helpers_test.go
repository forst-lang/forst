package devserver

import (
	"strings"
	"sync"
	"time"

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
