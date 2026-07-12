package nodert

import (
	"bytes"
	"io"
	"strings"
	"testing"

	logrus "github.com/sirupsen/logrus"
)

type writeRecorder struct {
	buf bytes.Buffer
}

func (w *writeRecorder) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func TestForwardChildOutput_copiesToDestination(t *testing.T) {
	pr, pw := io.Pipe()
	done := make(chan struct{})
	rec := &writeRecorder{}
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	go func() {
		forwardChildOutput(pr, rec, log, "stdout")
		close(done)
	}()

	if _, err := pw.Write([]byte("hello from node\n")); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	<-done

	if got := rec.buf.String(); got != "hello from node\n" {
		t.Fatalf("forwarded = %q", got)
	}
}

func TestForwardChildOutput_preservesAnsi(t *testing.T) {
	pr, pw := io.Pipe()
	done := make(chan struct{})
	rec := &writeRecorder{}
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	go func() {
		forwardChildOutput(pr, rec, log, "stdout")
		close(done)
	}()

	colored := "\x1b[32mgreen\x1b[0m\n"
	if _, err := pw.Write([]byte(colored)); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	<-done

	if got := rec.buf.String(); got != colored {
		t.Fatalf("forwarded = %q want %q", got, colored)
	}
}

func TestForwardChildOutput_logsAtDebug(t *testing.T) {
	pr, pw := io.Pipe()
	done := make(chan struct{})
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	var entries []string
	log.Hooks.Add(&testHook{onFire: func(e *logrus.Entry) {
		entries = append(entries, e.Message)
	}})

	go func() {
		forwardChildOutput(pr, io.Discard, log, "stderr")
		close(done)
	}()

	if _, err := pw.Write([]byte("warn line")); err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	<-done

	if len(entries) != 1 || !strings.Contains(entries[0], "warn line") {
		t.Fatalf("debug entries = %v", entries)
	}
}

type testHook struct {
	onFire func(*logrus.Entry)
}

func (h *testHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *testHook) Fire(e *logrus.Entry) error {
	if h.onFire != nil {
		h.onFire(e)
	}
	return nil
}
